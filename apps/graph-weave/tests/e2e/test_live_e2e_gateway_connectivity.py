import os
import pytest
import httpx
from src.adapters.ai_gateway_adapter import AIGatewayClient

def test_live_gateway_connectivity():
    """
    Live test to verify that the AI Gateway is reachable and responding.
    Skips if the gateway is not running localy or URL is not set.
    """
    gateway_url = os.getenv("AI_GATEWAY_URL", "http://localhost:8080/v1")
    
    # Check if gateway is alive
    health_url = gateway_url.replace("/v1", "/health")
    response = httpx.get(health_url, timeout=5.0)
    response.raise_for_status()

    client = AIGatewayClient(base_url=gateway_url)
    
    # Minimal chat completion to verify proxying
    # Use gpt-4.1 as configured in previous steps
    messages = [{"role": "user", "content": "Respond with the word 'ACKNOWLEDGE'"}]
    
    try:
        result = client.chat_completion(
            messages=messages,
            provider="github-copilot",
            model="gpt-4.1",
            max_tokens=5
        )
        
        content = result["choices"][0]["message"]["content"]
        assert "ACKNOWLEDGE" in content.upper()
        assert result["model"].startswith("gpt-4.1")
    except httpx.HTTPStatusError as e:
        pytest.fail(f"Gateway returned error status {e.response.status_code}: {e.response.text}")
    except Exception as e:
        pytest.fail(f"Gateway connection failed: {str(e)}")
