import httpx
import logging
import json
from typing import Any, Dict, Optional
from urllib.parse import parse_qsl, urlencode, urlparse, urlunparse
from .html_utils import extract_text_from_html

logger = logging.getLogger(__name__)

def fetch_wordpress_post(
    client: httpx.Client,
    original_url: str,
    headers: Dict[str, str],
    max_size_bytes: int
) -> Dict[str, Any]:
    api_url = wordpress_posts_api_url(original_url)
    if not api_url:
        return {"success": False, "error": "URL is not a WordPress post permalink", "url": original_url}

    fallback_errors = []
    try:
        response = client.get(api_url, headers={**headers, "Accept": "application/json"})
        response.raise_for_status()
        return wordpress_response_to_result(response, original_url, "wordpress_rest", max_size_bytes)
    except Exception as e:
        fallback_errors.append(f"WordPress REST fallback failed: {e}")

    wordpress_com = wordpress_com_public_api_url(original_url)
    if wordpress_com:
        try:
            response = client.get(wordpress_com, headers={**headers, "Accept": "application/json"})
            response.raise_for_status()
            return wordpress_response_to_result(response, original_url, "wordpress_com_public_api", max_size_bytes)
        except Exception as e:
            fallback_errors.append(f"WordPress.com public API fallback failed: {e}")

    return {"success": False, "error": "; ".join(fallback_errors), "url": original_url}

def wordpress_response_to_result(
    response: httpx.Response,
    original_url: str,
    source: str,
    max_size_bytes: int
) -> Dict[str, Any]:
    content_len = len(response.content)
    if content_len > max_size_bytes:
        return {
            "success": False,
            "error": f"WordPress response size ({content_len} bytes) exceeds limit ({max_size_bytes} bytes).",
            "url": original_url
        }

    try:
        payload = response.json()
    except ValueError as e:
        return {"success": False, "error": f"WordPress fallback returned invalid JSON: {e}", "url": original_url}

    post = payload[0] if isinstance(payload, list) and payload else payload
    if not isinstance(post, dict) or not post:
        return {"success": False, "error": "WordPress REST fallback returned no posts", "url": original_url}

    title_html = nested_rendered(post, "title")
    content_html = nested_rendered(post, "content") or nested_rendered(post, "excerpt")
    title = extract_text_from_html(title_html)
    body = extract_text_from_html(content_html)
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

def wordpress_posts_api_url(url: str) -> Optional[str]:
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

def wordpress_com_public_api_url(url: str) -> Optional[str]:
    parsed = urlparse(url)
    if not parsed.netloc.endswith("wordpress.com"):
        return None

    parts = [part for part in parsed.path.split("/") if part]
    if len(parts) < 4 or not all(part.isdigit() for part in parts[:3]):
        return None

    slug = parts[3]
    return f"https://public-api.wordpress.com/rest/v1.1/sites/{parsed.netloc}/posts/slug:{slug}"

def nested_rendered(value: Dict[str, Any], key: str) -> str:
    nested = value.get(key)
    if isinstance(nested, dict):
        rendered = nested.get("rendered")
        return rendered if isinstance(rendered, str) else ""
    return nested if isinstance(nested, str) else ""
