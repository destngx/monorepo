import os
import logging
import json
import warnings
from typing import Iterator
from src.app_logging import get_logger
from typing import Dict, Any, List, Optional, Union
import httpx
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception

logger = get_logger(__name__)

RETRYABLE_GATEWAY_STATUSES = {429, 502, 503, 504}


class AIGatewayResponsesError(RuntimeError):
    """Raised when a Responses request fails after its SSE stream has started."""


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

    def messages(
        self,
        messages: List[Dict[str, str]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.7,
        max_tokens: int = 8000,
        stream: bool = False,
        reasoning_effort: Optional[str] = None,
        use_responses: bool = True,
        output_schema: Optional[Dict[str, Any]] = None,
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
        if use_responses:
            events = list(self.messages_stream(
                messages=messages,
                provider=provider,
                model=model,
                tools=tools,
                max_tokens=max_tokens,
                reasoning_effort=reasoning_effort,
                output_schema=output_schema,
            ))
            body = "\n".join("data: " + json.dumps(event) for event in events)
            result = self._chat_response_from_responses_stream(body, model)
            message = result["choices"][0]["message"]
            if output_schema and not message.get("content"):
                raise AIGatewayResponsesError(
                    "Responses stream completed without structured output"
                )
            return result
        handler = self.chat_completion
        return handler(
            messages=messages,
            provider=provider,
            model=model,
            tools=tools,
            max_tokens=max_tokens,
            stream=stream,
            reasoning_effort=reasoning_effort,
        )

    def messages_stream(
        self,
        messages: List[Dict[str, Any]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        max_tokens: int = 8000,
        reasoning_effort: Optional[str] = None,
        output_schema: Optional[Dict[str, Any]] = None,
    ) -> Iterator[Dict[str, Any]]:
        """Yield native Responses SSE events without buffering the completion.

        Consumers must accumulate output text/tool arguments and validate the
        assembled result after receiving ``response.completed``.
        """
        instructions, input_items = self._responses_input(messages)
        payload: Dict[str, Any] = {
            "model": model, "input": input_items, "max_output_tokens": max_tokens,
            "stream": True, "store": False,
        }
        if instructions:
            payload["instructions"] = instructions
        if tools:
            payload["tools"] = self._responses_tools(tools)
        if reasoning_effort:
            payload["reasoning"] = {"effort": reasoning_effort}
        if output_schema:
            payload["text"] = {"format": {"type": "json_schema", "name": "graph_weave_output", "schema": output_schema, "strict": True}}

        url = f"{self.base_url}/responses"
        headers = {"Content-Type": "application/json", "X-AI-Provider": provider}
        with httpx.Client(timeout=self.timeout) as client:
            with client.stream("POST", url, headers=headers, json=payload) as response:
                if response.status_code != 200:
                    response.raise_for_status()
                for line in response.iter_lines():
                    if isinstance(line, bytes):
                        line = line.decode("utf-8", errors="replace")
                    if not line or not line.startswith("data:"):
                        continue
                    data = line[5:].lstrip()
                    if data == "[DONE]":
                        continue
                    try:
                        event = json.loads(data)
                    except json.JSONDecodeError:
                        logger.warning("Ignoring malformed Responses SSE event")
                        continue
                    self._raise_for_responses_event(event)
                    yield event

    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=2, max=10),
        retry=retry_if_exception(is_retryable_gateway_error),
        reraise=True,
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
        reasoning_effort: Optional[str] = None,
        output_schema: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Use the legacy OpenAI-compatible ``/chat/completions`` endpoint.

        Warning: Chat Completions is a compatibility path and will be deprecated
        in the future. Prefer :meth:`messages` or :meth:`responses`.
        """
        warnings.warn(
            "chat_completion() and /chat/completions are deprecated; use messages() or messages_stream().",
            DeprecationWarning,
            stacklevel=2,
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
    def responses(
        self,
        messages: List[Dict[str, Any]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        max_tokens: int = 8000,
        stream: bool = False,
        reasoning_effort: Optional[str] = None,
        output_schema: Optional[Dict[str, Any]] = None,
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
            # Codex's Responses endpoint only supports SSE. We consume the
            # stream here and retain Graph Weave's synchronous client contract.
            "stream": True,
            "store": False,
        }
        if instructions:
            payload["instructions"] = instructions
        if tools:
            payload["tools"] = self._responses_tools(tools)
        if reasoning_effort:
            payload["reasoning"] = {"effort": reasoning_effort}
        if output_schema:
            payload["text"] = {
                "format": {
                    "type": "json_schema",
                    "name": "graph_weave_output",
                    "schema": output_schema,
                    "strict": True,
                }
            }

        logger.debug(f"Sending Responses request to AI Gateway: {url} (Provider: {provider}, Model: {model})")
        try:
            with httpx.Client(timeout=self.timeout) as client:
                response = client.post(url, headers=headers, json=payload)
                if response.status_code != 200:
                    logger.error(f"AI Gateway Responses error ({response.status_code}): {response.text}")
                    response.raise_for_status()
                return self._chat_response_from_responses_stream(response.text, model)
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

    @classmethod
    def _chat_response_from_responses_stream(cls, body: str, fallback_model: str) -> Dict[str, Any]:
        content = ""
        completed_text = ""
        calls: Dict[str, Dict[str, Any]] = {}
        completed_response: Optional[Dict[str, Any]] = None

        for line in body.splitlines():
            if not line.startswith("data: "):
                continue
            payload = line.removeprefix("data: ")
            if payload == "[DONE]":
                continue
            try:
                event = json.loads(payload)
            except json.JSONDecodeError:
                continue
            event_type = event.get("type")
            cls._raise_for_responses_event(event)
            if event_type == "response.completed":
                completed_response = event.get("response")
                continue
            if event_type == "response.output_text.delta":
                content += event.get("delta", "")
                continue
            if event_type == "response.output_text.done":
                completed_text = event.get("text", "")
                continue
            if event_type in {"response.output_item.added", "response.output_item.done"}:
                item = event.get("item") or {}
                if item.get("type") == "message":
                    completed_text = "".join(
                        part.get("text", "")
                        for part in item.get("content", [])
                        if part.get("type") in {"output_text", "text"}
                    ) or completed_text
                    continue
                if item.get("type") != "function_call":
                    continue
                call_id = item.get("call_id") or item.get("id", "")
                call = calls.setdefault(call_id, {"id": call_id, "type": "function", "function": {"name": "", "arguments": ""}})
                call["function"]["name"] = item.get("name", call["function"]["name"])
                if item.get("arguments"):
                    call["function"]["arguments"] = item["arguments"]
                continue
            if event_type not in {"response.function_call_arguments.delta", "response.function_call_arguments.done"}:
                continue
            call_id = event.get("call_id", "")
            call = calls.setdefault(call_id, {"id": call_id, "type": "function", "function": {"name": event.get("name", ""), "arguments": ""}})
            if event.get("name"):
                call["function"]["name"] = event["name"]
            if event_type == "response.function_call_arguments.done" and event.get("arguments"):
                call["function"]["arguments"] = event["arguments"]
            else:
                call["function"]["arguments"] += event.get("delta", "")

        if completed_response and (completed_response.get("output") or completed_response.get("output_text")):
            return cls._chat_response_from_responses(completed_response, fallback_model)
        if not content:
            content = completed_text
        message: Dict[str, Any] = {"role": "assistant", "content": content}
        if calls:
            message["tool_calls"] = list(calls.values())
        result = {
            "id": (completed_response or {}).get("id", ""),
            "object": "chat.completion",
            "model": (completed_response or {}).get("model", fallback_model),
            "choices": [{"index": 0, "message": message, "finish_reason": "tool_calls" if calls else "stop"}],
            "usage": (completed_response or {}).get("usage", {}),
        }
        return result

    @staticmethod
    def _raise_for_responses_event(event: Dict[str, Any]) -> None:
        """Raise for both native Responses failures and gateway SSE errors.

        A streaming HTTP response is already committed as 200 before an
        upstream failure can occur, so the error must be detected in-band.
        """
        event_type = event.get("type")
        if event_type not in {"error", "response.failed"} and not (
            not event_type and event.get("error")
        ):
            return

        error: Any = event.get("error")
        response = event.get("response")
        if isinstance(response, dict) and response.get("error"):
            error = response["error"]
        if isinstance(error, dict):
            message = error.get("message") or error.get("code") or json.dumps(error)
        else:
            message = str(error or event.get("message") or event)
        raise AIGatewayResponsesError(f"AI Gateway Responses stream failed: {message}")
