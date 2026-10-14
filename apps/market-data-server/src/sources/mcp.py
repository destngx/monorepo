"""
Minimal MCP client over Streamable HTTP for sources that publish market data
as remote MCP servers (MARKET_PLATFORM.md section 2.5). It only needs
`initialize` and `tools/call`, so it speaks JSON-RPC over httpx directly
rather than pulling in the `mcp` SDK.
"""

import asyncio
import itertools
import json
from typing import Any, Dict, Optional

import httpx

from .http import StatusError

PROTOCOL_VERSION = "2025-06-18"
SESSION_HEADER = "mcp-session-id"


class McpError(Exception):
    """A JSON-RPC error, or an answer that is not a usable JSON-RPC result."""


def _decode(response: httpx.Response, request_id: int) -> Dict[str, Any]:
    """The JSON-RPC message answering request_id, from a JSON or an SSE body."""
    if "text/event-stream" in response.headers.get("content-type", ""):
        messages = []
        for line in response.text.splitlines():
            if line.startswith("data:"):
                try:
                    messages.append(json.loads(line[5:].strip()))
                except ValueError as e:
                    raise McpError(f"unreadable event: {e}") from e
    else:
        try:
            body = response.json()
        except ValueError as e:
            raise McpError(f"unreadable answer: {e}") from e
        messages = body if isinstance(body, list) else [body]
    for message in messages:
        if isinstance(message, dict) and message.get("id") == request_id:
            if "error" in message:
                error = message["error"] or {}
                raise McpError(f"{error.get('code')}: {error.get('message')}")
            if not isinstance(message.get("result"), dict):
                raise McpError("answer without a result")
            return message["result"]
    raise McpError(f"no answer to request {request_id}")


class McpClient:
    """
    One MCP endpoint (`path` on `client`'s base URL). The session is opened
    lazily, once, and reopened when the server forgets it (404).
    """

    def __init__(self, client: httpx.AsyncClient, source: str, path: str):
        self._client = client
        self._source = source
        self._path = path
        self._ids = itertools.count(1)
        self._lock = asyncio.Lock()
        self._ready = False
        self._session: Optional[str] = None

    async def _post(self, payload: Dict[str, Any]) -> httpx.Response:
        headers = {"Accept": "application/json, text/event-stream"}
        if self._session:
            headers[SESSION_HEADER] = self._session
        return await self._client.post(self._path, json=payload, headers=headers)

    async def _rpc(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        request_id = next(self._ids)
        response = await self._post({"jsonrpc": "2.0", "id": request_id, "method": method, "params": params})
        if not response.is_success:
            raise StatusError(self._source, response.status_code)
        if method == "initialize" and response.headers.get(SESSION_HEADER):
            self._session = response.headers[SESSION_HEADER]
        return _decode(response, request_id)

    async def connect(self) -> Dict[str, Any]:
        """Open a session; returns the server's `initialize` result."""
        async with self._lock:
            self._session = None
            result = await self._rpc(
                "initialize",
                {"protocolVersion": PROTOCOL_VERSION, "capabilities": {}, "clientInfo": {"name": "market-data-server", "version": "2"}},
            )
            response = await self._post({"jsonrpc": "2.0", "method": "notifications/initialized"})
            if not response.is_success:
                raise StatusError(self._source, response.status_code)
            self._ready = True
            return result

    async def call_tool(self, name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """The `tools/call` result: `{content: [...], isError?}`."""
        if not self._ready:
            await self.connect()
        try:
            return await self._rpc("tools/call", {"name": name, "arguments": arguments})
        except StatusError as e:
            if e.status != 404 or not self._session:
                raise
        # The server dropped our session: open a new one and retry once.
        await self.connect()
        return await self._rpc("tools/call", {"name": name, "arguments": arguments})
