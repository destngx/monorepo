import pytest
from unittest.mock import MagicMock, patch
import httpx
from src.adapters.web_tool import WebTool

@pytest.fixture
def web_tool():
    return WebTool()

def test_fetch_invalid_protocol(web_tool):
    result = web_tool.fetch("ftp://example.com")
    assert result["success"] is False
    assert "Invalid URL protocol" in result["error"]

def test_default_headers_use_curl_like_identity(web_tool):
    headers = web_tool._default_headers()

    assert headers["User-Agent"].startswith("curl/")
    assert headers["Accept"] == "*/*"

@patch("httpx.Client")
def test_fetch_success_html(mock_client_class, web_tool):
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client
    
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.content = b"<html><script>alert(1)</script><body><h1>Hello</h1><p>World</p></body></html>"
    mock_response.text = mock_response.content.decode()
    mock_response.headers = {"Content-Type": "text/html"}
    mock_response.url = "https://example.com"
    
    mock_client.get.return_value = mock_response
    
    result = web_tool.fetch("https://example.com")
    
    assert result["success"] is True
    assert result["status_code"] == 200
    # Text extraction check: script should be removed, tags removed, whitespace cleaned
    assert result["content"] == "Hello World"
    assert result["content_type"] == "text/html"

@patch("httpx.Client")
def test_fetch_success_json(mock_client_class, web_tool):
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client
    
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.content = b'{"status": "ok"}'
    mock_response.text = mock_response.content.decode()
    mock_response.headers = {"Content-Type": "application/json"}
    mock_response.url = "https://api.example.com"
    
    mock_client.get.return_value = mock_response
    
    result = web_tool.fetch("https://api.example.com")
    
    assert result["success"] is True
    assert result["content"] == '{"status": "ok"}'
    assert result["content_type"] == "application/json"

@patch("httpx.Client")
def test_fetch_size_limit(mock_client_class):
    tool = WebTool(max_size_bytes=10)
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client
    
    mock_response = MagicMock()
    mock_response.content = b"A" * 100
    mock_response.status_code = 200
    
    mock_client.get.return_value = mock_response
    
    result = tool.fetch("https://example.com")
    assert result["success"] is False
    assert "exceeds limit" in result["error"]

@patch("httpx.Client")
def test_fetch_timeout(mock_client_class, web_tool):
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client
    mock_client.get.side_effect = httpx.TimeoutException("Timeout")
    
    result = web_tool.fetch("https://example.com")
    assert result["success"] is False
    assert "timed out" in result["error"]

@patch("httpx.Client")
def test_fetch_wordpress_rest_fallback_on_403(mock_client_class, web_tool):
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client

    forbidden_response = MagicMock()
    forbidden_response.status_code = 403
    forbidden_response.reason_phrase = "Forbidden"
    forbidden_response.raise_for_status.side_effect = httpx.HTTPStatusError(
        "Forbidden",
        request=MagicMock(),
        response=forbidden_response,
    )

    rest_response = MagicMock()
    rest_response.status_code = 200
    rest_response.content = b'[{"title":{"rendered":"Post Title"},"content":{"rendered":"<p>Hello &amp; world.</p>"}}]'
    rest_response.headers = {"Content-Type": "application/json"}
    rest_response.url = "https://example.wordpress.com/wp-json/wp/v2/posts?slug=post-title"
    rest_response.json.return_value = [
        {
            "title": {"rendered": "Post Title"},
            "content": {"rendered": "<p>Hello &amp; world.</p>"},
        }
    ]

    mock_client.get.side_effect = [forbidden_response, rest_response]

    result = web_tool.fetch("https://example.wordpress.com/2010/11/06/post-title/")

    assert result["success"] is True
    assert result["source"] == "wordpress_rest"
    assert result["content"] == "Post Title\n\nHello & world."
    assert "fallback_reason" in result

@patch("httpx.Client")
def test_fetch_wordpress_com_public_api_fallback_when_local_rest_is_blocked(mock_client_class, web_tool):
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client

    forbidden_response = MagicMock()
    forbidden_response.status_code = 403
    forbidden_response.reason_phrase = "Forbidden"
    forbidden_response.raise_for_status.side_effect = httpx.HTTPStatusError(
        "Forbidden",
        request=MagicMock(),
        response=forbidden_response,
    )

    local_rest_response = MagicMock()
    local_rest_response.status_code = 403
    local_rest_response.reason_phrase = "Forbidden"
    local_rest_response.raise_for_status.side_effect = httpx.HTTPStatusError(
        "Forbidden",
        request=MagicMock(),
        response=local_rest_response,
    )

    public_api_response = MagicMock()
    public_api_response.status_code = 200
    public_api_response.content = b'{"title":"Post Title","content":"<p>Hello &amp; world.</p>"}'
    public_api_response.headers = {"Content-Type": "application/json"}
    public_api_response.url = "https://public-api.wordpress.com/rest/v1.1/sites/example.wordpress.com/posts/slug:post-title"
    public_api_response.json.return_value = {
        "title": "Post Title",
        "content": "<p>Hello &amp; world.</p>",
    }

    mock_client.get.side_effect = [
        forbidden_response,
        local_rest_response,
        public_api_response,
    ]

    result = web_tool.fetch("https://example.wordpress.com/2010/11/06/post-title/")

    assert result["success"] is True
    assert result["source"] == "wordpress_com_public_api"
    assert result["content"] == "Post Title\n\nHello & world."

def test_wordpress_posts_api_url(web_tool):
    api_url = web_tool._wordpress_posts_api_url(
        "https://agilewarrior.wordpress.com/2010/11/06/the-agile-inception-deck/"
    )

    assert api_url.startswith("https://agilewarrior.wordpress.com/wp-json/wp/v2/posts?")
    assert "slug=the-agile-inception-deck" in api_url
    assert "_fields=link%2Ctitle%2Ccontent%2Cexcerpt" in api_url

def test_wordpress_com_public_api_url(web_tool):
    api_url = web_tool._wordpress_com_public_api_url(
        "https://agilewarrior.wordpress.com/2010/11/06/the-agile-inception-deck/"
    )

    assert api_url == (
        "https://public-api.wordpress.com/rest/v1.1/sites/"
        "agilewarrior.wordpress.com/posts/slug:the-agile-inception-deck"
    )

def test_extract_text_from_html(web_tool):
    html = """
    <html>
        <head><title>Ignore</title></head>
        <style>body { color: red; }</style>
        <body>
            <header>Header</header>
            <nav>Nav</nav>
            <main>
                <h1>Title</h1>
                <p>Paragraph with &nbsp; space and &lt;tag&gt;.</p>
                <script>console.log('test')</script>
            </main>
            <footer>Footer</footer>
        </body>
    </html>
    """
    # Note: Our simple extractor removes header, footer, nav, script, style, title
    # And keeps the rest
    text = web_tool._extract_text_from_html(html)
    assert "Ignore" not in text
    assert "body {" not in text
    assert "Header" not in text
    assert "Nav" not in text
    assert "Footer" not in text
    assert "Title" in text
    assert "Paragraph with space and <tag>." in text
    assert "console.log" not in text
