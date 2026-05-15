import os
import logging
from src.app_logging import get_logger
from typing import Dict, Any, List, Optional, Union
import httpx
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception

logger = get_logger(__name__)

RETRYABLE_GATEWAY_STATUSES = {429, 502, 503, 504}


def is_retryable_gateway_error(exc: BaseException) -> bool:
    if isinstance(exc, (httpx.ConnectTimeout, httpx.ReadTimeout, httpx.RemoteProtocolError)):
        return True
    if isinstance(exc, httpx.HTTPStatusError):
        return exc.response.status_code in RETRYABLE_GATEWAY_STATUSES
    return False

class AIGatewayClient:
    """
    Client for interacting with the AI Gateway in Proxy Mode.
    Follows OpenAI-compatible chat completion API.
    """

    def __init__(self, base_url: Optional[str] = None):
        """
        Initialize the AI Gateway client.
        
        Args:
            base_url: The base URL for the AI Gateway (e.g., http://localhost:8080/v1)
        """
        self.base_url = (base_url or os.getenv("AI_GATEWAY_URL", "http://localhost:8080/v1")).rstrip("/")
        self.timeout = 600.0

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
        retry=retry_if_exception(is_retryable_gateway_error),
        reraise=True
    )
    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.7,
        max_tokens: int = 8000,
        stream: bool = False,
        reasoning_effort: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Send a chat completion request to the AI Gateway.
        
        Args:
            messages: List of message objects (role, content)
            provider: The backend provider to use (X-AI-Provider header)
            model: The model name to request
            tools: Optional list of tool definitions
            temperature: sampling temperature
            max_tokens: maximum tokens to generate
            stream: whether to stream the response (currently sync only)
            reasoning_effort: optional reasoning effort (e.g., 'low', 'medium', 'high')
            
        Returns:
            OpenAI-compatible response dictionary
        """
        url = f"{self.base_url}/chat/completions"
        
        headers = {
            "Content-Type": "application/json",
            "X-AI-Provider": provider
        }
        
        payload: Dict[str, Any] = {
            "model": model,
            "messages": messages,
            "temperature": temperature,
            "max_tokens": max_tokens,
            "stream": stream
        }
        
        if tools:
            payload["tools"] = tools
            
        if reasoning_effort:
            payload["reasoning_effort"] = reasoning_effort

        logger.debug(f"Sending request to AI Gateway: {url} (Provider: {provider}, Model: {model})")
        
        try:
            with httpx.Client(timeout=self.timeout) as client:
                response = client.post(url, headers=headers, json=payload)
                
                if response.status_code != 200:
                    logger.error(f"AI Gateway error ({response.status_code}): {response.text}")
                    response.raise_for_status()
                
                return response.json()
                
        except httpx.RequestError as exc:
            logger.error(f"An error occurred while requesting {exc.request.url!r}: {exc}")
            raise
        except Exception as exc:
            logger.error(f"Unexpected error during AI Gateway call: {exc}")
            raise
