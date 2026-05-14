import pytest
from unittest.mock import MagicMock, patch
import httpx
from src.adapters.mcp.tools.web import WebTool

@pytest.fixture
def web_tool():
    return WebTool()

def test_fetch_invalid_protocol(web_tool):
    result = web_tool.fetch("ftp://example.com")
    assert result["success"] is False
    assert "Invalid URL protocol" in result["error"]

def test_default_headers_use_browser_like_identity(web_tool):
    headers = web_tool._default_headers()

    assert "Mozilla/5.0" in headers["User-Agent"]
    assert "text/html" in headers["Accept"]

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
    assert result["content"] == "Hello\nWorld"
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
def test_fetch_retries_on_403_with_fallback_ua(mock_client_class, web_tool):
    """When the first UA strategy gets 403, retry with the next."""
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client

    forbidden_response = MagicMock()
    forbidden_response.status_code = 403
    forbidden_response.reason_phrase = "Forbidden"

    success_response = MagicMock()
    success_response.status_code = 200
    success_response.content = b"<html><body>OK</body></html>"
    success_response.text = success_response.content.decode()
    success_response.headers = {"Content-Type": "text/html"}
    success_response.url = "https://example.com"

    # First call (browser UA) raises 403, second call (curl UA) succeeds.
    def _get_side_effect(url, headers):
        resp = MagicMock()
        if "Mozilla" in headers.get("User-Agent", ""):
            resp.status_code = 403
            resp.reason_phrase = "Forbidden"
            resp.raise_for_status.side_effect = httpx.HTTPStatusError(
                "403", request=MagicMock(), response=forbidden_response
            )
        else:
            resp.status_code = 200
            resp.content = b"<html><body>OK</body></html>"
            resp.text = resp.content.decode()
            resp.headers = {"Content-Type": "text/html"}
            resp.url = "https://example.com"
            resp.raise_for_status.return_value = None
        return resp

    mock_client.get.side_effect = _get_side_effect

    result = web_tool.fetch("https://example.com")
    assert result["success"] is True
    assert mock_client.get.call_count == 2

@patch("httpx.Client")
def test_fetch_returns_error_when_all_strategies_fail(mock_client_class, web_tool):
    """When all UA strategies return 403, report the error."""
    mock_client = MagicMock()
    mock_client_class.return_value.__enter__.return_value = mock_client

    forbidden_response = MagicMock()
    forbidden_response.status_code = 403
    forbidden_response.reason_phrase = "Forbidden"

    mock_resp = MagicMock()
    mock_resp.raise_for_status.side_effect = httpx.HTTPStatusError(
        "403", request=MagicMock(), response=forbidden_response
    )
    mock_client.get.return_value = mock_resp

    result = web_tool.fetch("https://example.com")
    assert result["success"] is False
    assert "403" in result["error"]
    # Should have tried both strategies
    assert mock_client.get.call_count == 2

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
    # The new extractor might add newlines/extra whitespace between words but the content should be there
    assert "Paragraph with space and <tag>." in text
    assert "console.log" not in text
