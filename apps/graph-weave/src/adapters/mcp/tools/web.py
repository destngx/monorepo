import httpx
import logging
from typing import Dict, Any, Optional
from src.app_logging import get_logger
from ..infra.html_utils import extract_text_from_html
from ..infra.exceptions import ToolExecutionError

logger = get_logger(__name__)

def handle_fetch(web_tool: Any, url: str, method: str = "GET", headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    """Execute web fetch tool."""
    try:
        result = web_tool.fetch(url, method=method, headers=headers)
        return {
            "tool": "fetch",
            "url": url,
            "status": "success" if result.get("success", False) else "error",
            **result
        }
    except Exception as e:
        raise ToolExecutionError(f"Failed to fetch URL '{url}': {str(e)}")

class WebToolError(Exception):
    """Exception raised for errors in the WebTool."""
    pass

class WebTool:
    """
    Safely fetches web content with size limits and timeouts.
    """

    # Browser-like UA accepted by most sites (StackOverflow, GitHub, etc.).
    _BROWSER_UA = (
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/126.0.0.0 Safari/537.36"
    )
    # Minimal curl UA bypasses JS-challenge systems (WordPress, some CDNs).
    _CURL_UA = "curl/8.7.1"

    # Ordered list of (user-agent, headers) strategies to try.
    _UA_STRATEGIES = [
        (
            _BROWSER_UA,
            {
                "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
                "Accept-Language": "en-US,en;q=0.9",
                "Accept-Encoding": "gzip, deflate, br",
                "Cache-Control": "no-cache",
            },
        ),
        (
            _CURL_UA,
            {
                "Accept": "*/*",
            },
        ),
    ]

    def __init__(self, max_size_bytes: int = 1_000_000):
        self.max_size_bytes = max_size_bytes

    def fetch(
        self, 
        url: str, 
        method: str = "GET", 
        headers: Optional[Dict[str, str]] = None,
        timeout: int = 15
    ) -> Dict[str, Any]:
        """
        Fetch content from a URL.

        Uses a browser-like User-Agent by default. If a 403 is returned,
        automatically retries with a curl-like UA (helps with CDN challenge
        pages that flag browser UAs without JS support).
        """
        if not url.startswith(("http://", "https://")):
            return {
                "success": False,
                "error": "Invalid URL protocol. Only http and https are allowed.",
                "url": url
            }

        last_error: Optional[httpx.HTTPStatusError] = None

        for ua, default_hdrs in self._UA_STRATEGIES:
            request_headers = {"User-Agent": ua, **default_hdrs}
            if headers:
                request_headers.update(headers)

            try:
                with httpx.Client(follow_redirects=True, timeout=timeout) as client:
                    if method.upper() == "GET":
                        response = client.get(url, headers=request_headers)
                    elif method.upper() == "POST":
                        response = client.post(url, headers=request_headers)
                    else:
                        return {"success": False, "error": f"Unsupported method: {method}"}

                    response.raise_for_status()
                    return self._response_to_result(response)

            except httpx.HTTPStatusError as e:
                last_error = e
                if e.response.status_code == 403:
                    logger.debug(
                        f"Got 403 with UA '{ua[:20]}…', retrying with next strategy"
                    )
                    continue
                # Non-403 HTTP errors: no point retrying with a different UA.
                break
            except httpx.TimeoutException:
                logger.warning(f"Timeout fetching {url}")
                return {"success": False, "error": "Request timed out", "url": url}
            except Exception as e:
                logger.error(f"Error fetching {url}: {e}")
                return {"success": False, "error": str(e), "url": url}

        # All strategies exhausted or a non-retryable HTTP error occurred.
        if last_error is not None:
            logger.warning(f"HTTP error fetching {url}: {last_error}")
            return {
                "success": False,
                "error": f"HTTP {last_error.response.status_code}: {last_error.response.reason_phrase}",
                "url": url
            }
        # Should not reach here, but guard defensively.
        return {"success": False, "error": "Unknown fetch error", "url": url}

    def _default_headers(self) -> Dict[str, str]:
        """Return browser-like default headers (primary strategy)."""
        ua, hdrs = self._UA_STRATEGIES[0]
        return {"User-Agent": ua, **hdrs}

    def _response_to_result(self, response: httpx.Response) -> Dict[str, Any]:
        content_len = len(response.content)
        if content_len > self.max_size_bytes:
            return {
                "success": False,
                "error": f"Response size ({content_len} bytes) exceeds limit ({self.max_size_bytes} bytes).",
                "url": str(response.url)
            }

        content_type = response.headers.get("Content-Type", "")
        text_content = response.text

        if "text/html" in content_type:
            cleaned_text = extract_text_from_html(text_content)
        else:
            cleaned_text = text_content

        return {
            "success": True,
            "url": str(response.url),
            "status_code": response.status_code,
            "content_type": content_type,
            "content": cleaned_text,
            "raw_length": content_len
        }

    def _extract_text_from_html(self, html: str) -> str:
        return extract_text_from_html(html)
