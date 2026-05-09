import sys
import os
import json
import asyncio

# Add src to path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))

from src.adapters.mcp_router import MCPRouter

def test_fetch():
    router = MCPRouter()
    
    # Test GET request to a public API or website
    url = "https://httpbin.org/get"
    print(f"Testing fetch (GET) for {url}...")
    
    result = router.execute_tool("fetch", {"url": url})
    print(f"Status: {result.get('status')}")
    if result.get('status') == 'success':
        print(f"Content length: {len(result.get('content', ''))}")
        print(f"Content preview: {result.get('content')[:100]}...")
    else:
        print(f"Error: {result.get('error')}")

    # Test HTML text extraction
    url_html = "https://example.com"
    print(f"\nTesting fetch (HTML Text Extraction) for {url_html}...")
    result_html = router.execute_tool("fetch", {"url": url_html})
    print(f"Status: {result_html.get('status')}")
    if result_html.get('status') == 'success':
        print(f"Extracted content: {result_html.get('content')}")
    else:
        print(f"Error: {result_html.get('error')}")

if __name__ == "__main__":
    test_fetch()
