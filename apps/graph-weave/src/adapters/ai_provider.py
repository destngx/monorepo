"""
AI providers for GraphWeave.

Includes the existing deterministic mock provider and a GitHub Copilot-backed
provider for the MVP integration path.
"""

from typing import Protocol, TypedDict, cast, Optional, Dict, Any
import os
import json
import time
import logging
import threading

import httpx


logger = logging.getLogger(__name__)


class AIProvider(Protocol):
    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str = "gpt-4",
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> dict[str, object]: ...


class CopilotMessage(TypedDict):
    content: str


class CopilotChoice(TypedDict):
    message: CopilotMessage


class CopilotUsage(TypedDict, total=False):
    total_tokens: int


class CopilotResponse(TypedDict):
    choices: list[CopilotChoice]
    usage: CopilotUsage
    model: str


class CopilotTokenCache(TypedDict):
    token: str
    expires_at: float


class MockAIProvider:
    """Mock AI provider that returns deterministic responses based on prompt type and content."""

    def __init__(self):
        """Initialize the mock AI provider."""
        self._call_count: int = 0

    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str = "gpt-4",
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> dict[str, object]:
        self._call_count += 1

        content = self._generate_response(system_prompt, user_prompt)

        return {
            "content": content,
            "tokens_used": self._estimate_tokens(content),
            "model": model,
            "call_count": self._call_count,
        }

    def _generate_response(self, system_prompt: str, user_prompt: str) -> str:
        system_lower = system_prompt.lower()
        user_lower = user_prompt.lower()
        combined = f"{system_lower} {user_lower}"

        if any(
            keyword in combined
            for keyword in ["stagnation", "repeat", "loop", "stuck", "progress"]
        ):
            return self._stagnation_response(user_prompt)

        if any(
            keyword in combined
            for keyword in [
                "synthesize",
                "synthesis",
                "analyze",
                "summarize",
                "combine",
                "merge",
            ]
        ):
            return self._synthesis_response(user_prompt)

        if any(
            keyword in combined
            for keyword in ["search", "research", "find", "lookup", "web"]
        ):
            return self._research_response(user_prompt)

        if any(keyword in combined for keyword in ["sql", "query", "database", "data"]):
            return self._sql_response(user_prompt)

        if any(
            keyword in combined
            for keyword in ["classify", "categorize", "route", "type", "kind"]
        ):
            return self._classification_response(user_prompt)

        return self._default_response(user_prompt)

    def _research_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "research_complete",
                "findings": [
                    "Found relevant Q3 2024 earnings transcripts",
                    "Located industry comparison data",
                    "Retrieved competitor performance metrics",
                ],
                "sources": [
                    "SEC EDGAR (10-Q filings)",
                    "Earnings call transcripts",
                    "Bloomberg terminal",
                ],
                "confidence": 0.92,
                "next_step": "synthesize_with_internal_data",
            },
            indent=2,
        )

    def _sql_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "query_complete",
                "query": "SELECT * FROM performance_metrics WHERE date >= '2024-07-01'",
                "rows": [
                    {
                        "date": "2024-09-30",
                        "revenue": 2850000,
                        "profit_margin": 0.18,
                    },
                    {
                        "date": "2024-08-31",
                        "revenue": 2720000,
                        "profit_margin": 0.17,
                    },
                ],
                "row_count": 3,
                "execution_time_ms": 145,
            },
            indent=2,
        )

    def _synthesis_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "synthesis_complete",
                "summary": "Q3 2024 showed strong revenue growth (+4.8%) and margin expansion (17-18%), with operational efficiency gains offsetting elevated commodity costs. Industry positioning remains competitive vs. peers.",
                "key_insights": [
                    "Revenue growth outpacing sector average",
                    "Operating leverage improving",
                    "Market share gains in core segments",
                ],
                "risks": [
                    "Supply chain volatility",
                    "FX headwinds in emerging markets",
                ],
                "confidence": 0.88,
            },
            indent=2,
        )

    def _classification_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "classification_complete",
                "classification": "financial_analysis",
                "confidence": 0.95,
                "sub_types": ["earnings_review", "competitor_analysis"],
                "requires_external_research": True,
                "requires_internal_data": True,
            },
            indent=2,
        )

    def _stagnation_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "stagnation_check_complete",
                "detected_stagnation": False,
                "repeated_intents": 0,
                "unique_steps_taken": 3,
                "recommendation": "continue_execution",
                "last_unique_step": "synthesize_findings",
            },
            indent=2,
        )

    def _default_response(self, user_prompt: str) -> str:
        return json.dumps(
            {
                "status": "completed",
                "message": "Task processed successfully",
                "result": "Mock response generated",
                "call_count": self._call_count,
            },
            indent=2,
        )

    def _estimate_tokens(self, content: str) -> int:
        return max(1, len(content) // 4)

    def reset(self) -> None:
        self._call_count = 0


class GitHubCopilotProvider:
    """GitHub Copilot provider with cached token management and retry logic."""

    _token_lock: threading.Lock = threading.Lock()

    def __init__(
        self,
        gh_token: str,
        base_url: str = "https://api.githubcopilot.com",
        max_retries: int = 2,
    ):
        self.gh_token: str = gh_token
        self.base_url: str = base_url.rstrip("/")
        self.max_retries: int = max_retries
        self._cached_token: Optional[CopilotTokenCache] = None

    def get_copilot_token(self) -> str:
        """Get or refresh GitHub Copilot token (cached for 5 mins)."""
        now = time.time()

        if self._cached_token and self._cached_token["expires_at"] > now + 300:
            return self._cached_token["token"]

        with self._token_lock:
            if self._cached_token and self._cached_token["expires_at"] > now + 300:
                return self._cached_token["token"]

            logger.debug("Fetching new Copilot token from GitHub API")
            response = httpx.get(
                "https://api.github.com/copilot_internal/v2/token",
                headers={
                    "Authorization": f"token {self.gh_token}",
                    "Accept": "application/json",
                },
                timeout=10,
            )
            response.raise_for_status()
            data = response.json()

            self._cached_token = {
                "token": data["token"],
                "expires_at": data["expires_at"],
            }
            logger.debug(f"Copilot token refreshed, expires at {data['expires_at']}")
            return data["token"]

    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str = "gpt-4.1",
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> dict[str, object]:
        """Call GitHub Copilot API with retry and error recovery."""
        last_error: Optional[Exception] = None

        for attempt in range(self.max_retries):
            try:
                copilot_token = self.get_copilot_token()

                payload: Dict[str, Any] = {
                    "model": model,
                    "messages": [
                        {"role": "system", "content": system_prompt},
                        {"role": "user", "content": user_prompt},
                    ],
                    "temperature": temperature,
                    "max_tokens": max_tokens,
                }
                payload = {k: v for k, v in payload.items() if v is not None}

                response = httpx.post(
                    f"{self.base_url}/chat/completions",
                    json=payload,
                    headers={
                        "Authorization": f"Bearer {copilot_token}",
                        "Editor-Version": "vscode/1.80.0",
                        "Editor-Plugin-Version": "copilot-chat/0.1.0",
                    },
                    timeout=30,
                )

                if response.status_code == 401:
                    self._cached_token = None
                    if attempt < self.max_retries - 1:
                        logger.warning("Token expired (401); refreshing and retrying")
                        continue
                    raise httpx.HTTPStatusError(
                        "Copilot token refresh failed",
                        request=response.request,
                        response=response,
                    )

                if response.status_code == 429:
                    wait_time = 2**attempt
                    if attempt < self.max_retries - 1:
                        logger.warning(
                            f"Rate limited; waiting {wait_time}s before retry"
                        )
                        time.sleep(wait_time)
                        continue

                response.raise_for_status()

                data = cast(CopilotResponse, response.json())

                if "choices" not in data or not data["choices"]:
                    raise ValueError("Invalid API response: no choices returned")

                content = data["choices"][0]["message"]["content"]
                return {
                    "content": content,
                    "tokens_used": data.get("usage", {}).get("total_tokens", 0),
                    "model": data.get("model", model),
                }

            except httpx.ConnectError as e:
                last_error = e
                if attempt < self.max_retries - 1:
                    wait_time = 2**attempt
                    logger.warning(
                        f"Connection error (attempt {attempt + 1}/{self.max_retries}); "
                        f"waiting {wait_time}s: {e}"
                    )
                    time.sleep(wait_time)
                else:
                    logger.error(
                        f"Failed to reach Copilot API after {self.max_retries} attempts"
                    )
                    raise ConnectionError(
                        f"Failed to reach Copilot API after {self.max_retries} attempts"
                    ) from e

            except httpx.TimeoutException as e:
                last_error = e
                if attempt < self.max_retries - 1:
                    logger.warning(
                        f"Request timeout (attempt {attempt + 1}/{self.max_retries}); retrying"
                    )
                    continue
                logger.error("Request timeout after all retries")
                raise

            except (json.JSONDecodeError, KeyError, ValueError) as e:
                logger.error(f"Invalid response from Copilot API: {e}", exc_info=True)
                raise ValueError(f"Copilot API returned invalid response: {e}") from e

        if last_error:
            raise last_error
        raise RuntimeError("Unexpected state: no error or valid response")


def create_ai_provider(use_github_provider: bool = False) -> AIProvider:
    """
    Create AI provider with fallback chain.

    Priority:
    1. GitHub Copilot (if GITHUB_TOKEN set, or use_github_provider=True)
    2. Mock provider (fallback for testing/dev)

    Args:
        use_github_provider: Force GitHub provider (for health checks)

    Returns:
        AIProvider instance
    """
    gh_token = os.getenv("GITHUB_TOKEN")

    # Use GitHub provider if token exists OR explicitly requested
    if gh_token or use_github_provider:
        if gh_token:
            logger.info("Creating GitHubCopilotProvider (GITHUB_TOKEN found)")
            try:
                return GitHubCopilotProvider(gh_token=gh_token)
            except Exception as e:
                logger.warning(
                    f"Failed to create GitHub provider: {e}, falling back to mock"
                )
                return MockAIProvider()
        else:
            logger.warning("GitHub provider requested but GITHUB_TOKEN not set")
            return MockAIProvider()

    logger.info("Creating MockAIProvider (no GitHub token, using mock mode)")
    return MockAIProvider()
