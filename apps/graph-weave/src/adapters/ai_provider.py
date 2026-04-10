"""
AI providers for GraphWeave.

Includes the existing deterministic mock provider and a GitHub Copilot-backed
provider for the MVP integration path.
"""

from typing import Protocol, TypedDict, cast
import os
import json

import httpx


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


class MockAIProvider:
    """Mock AI provider that returns deterministic responses based on prompt type and content."""

    def __init__(self):
        """Initialize the mock AI provider."""
        self._call_count = 0

    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str = "gpt-4",
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> dict[str, object]:
        self._call_count += 1

        # Determine response type based on prompt keywords
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
    def __init__(self, token: str, base_url: str = "https://api.githubcopilot.com"):
        self.token: str = token
        self.base_url: str = base_url.rstrip("/")

    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str = "gpt-4.1",
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> dict[str, object]:
        response = httpx.post(
            f"{self.base_url}/chat/completions",
            json={
                "model": model,
                "messages": [
                    {"role": "system", "content": system_prompt},
                    {"role": "user", "content": user_prompt},
                ],
                "temperature": temperature,
                "max_tokens": max_tokens,
            },
            headers={"Authorization": f"Bearer {self.token}"},
            timeout=30,
        )
        _ = response.raise_for_status()
        payload = cast(CopilotResponse, response.json())

        choices = payload["choices"]
        content = choices[0]["message"]["content"]
        return {
            "content": content,
            "tokens_used": payload.get("usage", {}).get("total_tokens", 0),
            "model": payload.get("model", model),
        }


def create_ai_provider(use_github_provider: bool = False) -> AIProvider:
    token = os.getenv("GITHUB_TOKEN")
    if token:
        return GitHubCopilotProvider(token=token)
    raise RuntimeError("GITHUB_TOKEN is required for GitHub Copilot provider")
