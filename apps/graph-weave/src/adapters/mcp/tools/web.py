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

    def __init__(self, max_size_bytes: int = 1_000_000):
        self.max_size_bytes = max_size_bytes
        # WordPress.com and some CDN challenge systems treat browser-like
        # user agents without a real browser runtime as suspicious. A
        # curl-like fetch identity is less likely to trigger JS challenges and
        # matches the CLI behavior users expect from this tool.
        self.user_agent = "curl/8.7.1"

    def fetch(
        self, 
        url: str, 
        method: str = "GET", 
        headers: Optional[Dict[str, str]] = None,
        timeout: int = 15
    ) -> Dict[str, Any]:
        """
        Fetch content from a URL.
        """
        if not url.startswith(("http://", "https://")):
            return {
                "success": False,
                "error": "Invalid URL protocol. Only http and https are allowed.",
                "url": url
            }

        request_headers = self._default_headers()
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
            logger.warning(f"HTTP error fetching {url}: {e}")
            return {
                "success": False,
                "error": f"HTTP {e.response.status_code}: {e.response.reason_phrase}",
                "url": url
            }
        except httpx.TimeoutException:
            logger.warning(f"Timeout fetching {url}")
            return {"success": False, "error": "Request timed out", "url": url}
        except Exception as e:
            logger.error(f"Error fetching {url}: {e}")
            return {"success": False, "error": str(e), "url": url}

    def _default_headers(self) -> Dict[str, str]:
        return {
            "User-Agent": self.user_agent,
            "Accept": "*/*",
        }

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
