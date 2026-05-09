import httpx
import re
from typing import Dict, Any, Optional
from src.app_logging import get_logger

logger = get_logger(__name__)

class WebToolError(Exception):
    """Exception raised for errors in the WebTool."""
    pass

class WebTool:
    """
    Safely fetches web content with size limits and timeouts.
    """

    def __init__(self, max_size_bytes: int = 1_000_000):
        self.max_size_bytes = max_size_bytes
        self.user_agent = (
            "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
            "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
        )

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

        request_headers = {"User-Agent": self.user_agent}
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

                # Check content size
                content_len = len(response.content)
                if content_len > self.max_size_bytes:
                    return {
                        "success": False,
                        "error": f"Response size ({content_len} bytes) exceeds limit ({self.max_size_bytes} bytes).",
                        "url": url
                    }

                content_type = response.headers.get("Content-Type", "")
                text_content = response.text

                # If HTML, perform simple text extraction
                if "text/html" in content_type:
                    cleaned_text = self._extract_text_from_html(text_content)
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

    def _extract_text_from_html(self, html: str) -> str:
        """
        Perform basic text extraction from HTML without external dependencies.
        Removes scripts, styles, and tags.
        """
        # Remove script and style elements
        text = re.sub(r'<(script|style|title|header|footer|nav).*?>.*?</\1>', '', html, flags=re.DOTALL | re.IGNORECASE)
        
        # Remove all other tags
        text = re.sub(r'<.*?>', ' ', text)
        
        # Replace entities
        text = text.replace("&nbsp;", " ").replace("&lt;", "<").replace("&gt;", ">").replace("&amp;", "&")
        
        # Clean up whitespace
        text = re.sub(r'\s+', ' ', text).strip()
        
        return text
