import httpx
import html as html_lib
import re
from urllib.parse import parse_qsl, urlencode, urlparse, urlunparse
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

                try:
                    response.raise_for_status()
                except httpx.HTTPStatusError as e:
                    if method.upper() == "GET" and e.response.status_code in {401, 403, 406, 429}:
                        fallback = self._fetch_wordpress_post(client, url, request_headers)
                        if fallback.get("success"):
                            fallback["fallback_reason"] = f"HTTP {e.response.status_code}: {e.response.reason_phrase}"
                            return fallback
                    raise

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

    def _fetch_wordpress_post(
        self,
        client: httpx.Client,
        original_url: str,
        headers: Dict[str, str],
    ) -> Dict[str, Any]:
        api_url = self._wordpress_posts_api_url(original_url)
        if not api_url:
            return {"success": False, "error": "URL is not a WordPress post permalink", "url": original_url}

        fallback_errors = []
        try:
            response = client.get(api_url, headers={**headers, "Accept": "application/json"})
            response.raise_for_status()
            return self._wordpress_response_to_result(response, original_url, "wordpress_rest")
        except Exception as e:
            fallback_errors.append(f"WordPress REST fallback failed: {e}")

        wordpress_com = self._wordpress_com_public_api_url(original_url)
        if wordpress_com:
            try:
                response = client.get(wordpress_com, headers={**headers, "Accept": "application/json"})
                response.raise_for_status()
                return self._wordpress_response_to_result(response, original_url, "wordpress_com_public_api")
            except Exception as e:
                fallback_errors.append(f"WordPress.com public API fallback failed: {e}")

        return {"success": False, "error": "; ".join(fallback_errors), "url": original_url}

    def _wordpress_response_to_result(
        self,
        response: httpx.Response,
        original_url: str,
        source: str,
    ) -> Dict[str, Any]:
        content_len = len(response.content)
        if content_len > self.max_size_bytes:
            return {
                "success": False,
                "error": f"WordPress response size ({content_len} bytes) exceeds limit ({self.max_size_bytes} bytes).",
                "url": original_url
            }

        try:
            payload = response.json()
        except ValueError as e:
            return {"success": False, "error": f"WordPress fallback returned invalid JSON: {e}", "url": original_url}

        post = payload[0] if isinstance(payload, list) and payload else payload
        if not isinstance(post, dict) or not post:
            return {"success": False, "error": "WordPress REST fallback returned no posts", "url": original_url}

        title_html = self._nested_rendered(post, "title")
        content_html = self._nested_rendered(post, "content") or self._nested_rendered(post, "excerpt")
        title = self._extract_text_from_html(title_html)
        body = self._extract_text_from_html(content_html)
        content = "\n\n".join(part for part in [title, body] if part)

        return {
            "success": True,
            "url": original_url,
            "status_code": response.status_code,
            "content_type": response.headers.get("Content-Type", "application/json"),
            "content": content,
            "raw_length": content_len,
            "source": source,
            "api_url": str(response.url),
        }

    def _wordpress_posts_api_url(self, url: str) -> Optional[str]:
        parsed = urlparse(url)
        parts = [part for part in parsed.path.split("/") if part]
        if len(parts) < 4 or not all(part.isdigit() for part in parts[:3]):
            return None

        slug = parts[3]
        query = dict(parse_qsl(parsed.query, keep_blank_values=True))
        query.update(
            {
                "slug": slug,
                "_fields": "link,title,content,excerpt",
                "per_page": "1",
            }
        )
        return urlunparse(
            (
                parsed.scheme,
                parsed.netloc,
                "/wp-json/wp/v2/posts",
                "",
                urlencode(query),
                "",
            )
        )

    def _wordpress_com_public_api_url(self, url: str) -> Optional[str]:
        parsed = urlparse(url)
        if not parsed.netloc.endswith("wordpress.com"):
            return None

        parts = [part for part in parsed.path.split("/") if part]
        if len(parts) < 4 or not all(part.isdigit() for part in parts[:3]):
            return None

        slug = parts[3]
        return f"https://public-api.wordpress.com/rest/v1.1/sites/{parsed.netloc}/posts/slug:{slug}"

    def _nested_rendered(self, value: Dict[str, Any], key: str) -> str:
        nested = value.get(key)
        if isinstance(nested, dict):
            rendered = nested.get("rendered")
            return rendered if isinstance(rendered, str) else ""
        return nested if isinstance(nested, str) else ""

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
        text = html_lib.unescape(text.replace("&nbsp;", " "))
        
        # Clean up whitespace
        text = re.sub(r'\s+', ' ', text).strip()
        
        return text
