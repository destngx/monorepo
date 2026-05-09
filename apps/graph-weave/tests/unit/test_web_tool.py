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
