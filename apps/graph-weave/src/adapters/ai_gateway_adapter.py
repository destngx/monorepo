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
        if provider == "openai":
            return self.responses_completion(
                messages=messages,
                provider=provider,
                model=model,
                tools=tools,
                max_tokens=max_tokens,
                stream=stream,
                reasoning_effort=reasoning_effort,
            )

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

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
        retry=retry_if_exception(is_retryable_gateway_error),
        reraise=True,
    )
    def responses_completion(
        self,
        messages: List[Dict[str, Any]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        max_tokens: int = 8000,
        stream: bool = False,
        reasoning_effort: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Call the native Responses endpoint and return a chat-compatible result.

        Graph Weave's nodes use Chat Completions message and tool-call shapes. This
        adapter keeps that internal contract while avoiding the gateway's additional
        Chat Completions-to-Responses translation for OpenAI requests.
        """
        url = f"{self.base_url}/responses"
        headers = {"Content-Type": "application/json", "X-AI-Provider": provider}
        instructions, input_items = self._responses_input(messages)
        payload: Dict[str, Any] = {
            "model": model,
            "input": input_items,
            "max_output_tokens": max_tokens,
            "stream": stream,
            "store": False,
        }
        if instructions:
            payload["instructions"] = instructions
        if tools:
            payload["tools"] = self._responses_tools(tools)
        if reasoning_effort:
            payload["reasoning"] = {"effort": reasoning_effort}

        logger.debug(f"Sending Responses request to AI Gateway: {url} (Provider: {provider}, Model: {model})")
        try:
            with httpx.Client(timeout=self.timeout) as client:
                response = client.post(url, headers=headers, json=payload)
                if response.status_code != 200:
                    logger.error(f"AI Gateway Responses error ({response.status_code}): {response.text}")
                    response.raise_for_status()
                return self._chat_response_from_responses(response.json(), model)
        except httpx.RequestError as exc:
            logger.error(f"An error occurred while requesting {exc.request.url!r}: {exc}")
            raise
        except Exception as exc:
            logger.error(f"Unexpected error during AI Gateway Responses call: {exc}")
            raise

    @staticmethod
    def _responses_input(messages: List[Dict[str, Any]]) -> tuple[str, List[Dict[str, Any]]]:
        instructions = []
        input_items = []
        for message in messages:
            role = message.get("role")
            content = message.get("content", "")
            if role == "system":
                if content:
                    instructions.append(content)
                continue
            if role == "tool":
                input_items.append({"type": "function_call_output", "call_id": message["tool_call_id"], "output": content})
                continue
            for call in message.get("tool_calls") or []:
                function = call.get("function", {})
                input_items.append({"type": "function_call", "call_id": call["id"], "name": function.get("name", ""), "arguments": function.get("arguments", "")})
            if content or not message.get("tool_calls"):
                input_items.append({"role": role, "content": content})
        return "\n".join(instructions), input_items

    @staticmethod
    def _responses_tools(tools: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        return [
            {"type": "function", **tool["function"]}
            for tool in tools
            if tool.get("type") == "function" and tool.get("function")
        ]

    @staticmethod
    def _chat_response_from_responses(response: Dict[str, Any], fallback_model: str) -> Dict[str, Any]:
        content = response.get("output_text", "")
        tool_calls = []
        for item in response.get("output", []):
            if item.get("type") == "function_call":
                tool_calls.append({
                    "id": item.get("call_id", ""),
                    "type": "function",
                    "function": {"name": item.get("name", ""), "arguments": item.get("arguments", "")},
                })
            for part in item.get("content", []):
                if part.get("type") in {"output_text", "text"}:
                    content += part.get("text", "")
        message: Dict[str, Any] = {"role": "assistant", "content": content}
        if tool_calls:
            message["tool_calls"] = tool_calls
        return {
            "id": response.get("id", ""),
            "object": "chat.completion",
            "model": response.get("model", fallback_model),
            "choices": [{"index": 0, "message": message, "finish_reason": "tool_calls" if tool_calls else "stop"}],
            "usage": response.get("usage", {}),
        }
