"""Shared request-and-decode helper for sources that call public JSON APIs."""

from typing import Any, Mapping, Optional

import httpx

# Bounds one upstream call so a slow source fails over fast.
DEFAULT_TIMEOUT = 10.0


def new_client(base_url: str, headers: Optional[Mapping[str, str]] = None, transport: Optional[httpx.AsyncBaseTransport] = None) -> httpx.AsyncClient:
    """One client per source; `transport` lets tests answer without a network."""
    if not (base_url or "").strip():
        raise ValueError("missing base URL")
    return httpx.AsyncClient(
        base_url=base_url.strip().rstrip("/"),
        headers={"Accept": "application/json", **(headers or {})},
        timeout=DEFAULT_TIMEOUT,
        transport=transport,
    )


# Error bodies larger than this are not read for an upstream error code.
MAX_ERROR_BODY = 64 * 1024


class StatusError(Exception):
    """
    A non-2xx answer. The body is not kept: upstream error pages are large and
    may echo the request. `code` is the `code` field of a small JSON error
    body (fmarket's own error codes), else None.
    """

    def __init__(self, source: str, status: int, code: Any = None):
        super().__init__(f"{source} request failed with status {status}")
        self.source = source
        self.status = status
        self.code = code


def _error_code(response: httpx.Response) -> Any:
    if "json" not in response.headers.get("content-type", "") or len(response.content) > MAX_ERROR_BODY:
        return None
    try:
        body = response.json()
    except ValueError:
        return None
    return body.get("code") if isinstance(body, dict) else None


async def request_json(
    client: httpx.AsyncClient,
    source: str,
    method: str,
    path: str,
    *,
    params: Optional[Mapping[str, Any]] = None,
    json: Any = None,
    headers: Optional[Mapping[str, str]] = None,
) -> Any:
    response = await client.request(method, path, params=params, json=json, headers=headers)
    if not response.is_success:
        raise StatusError(source, response.status_code, _error_code(response))
    try:
        return response.json()
    except ValueError as e:
        raise ValueError(f"{source}: decode response: {e}") from e
