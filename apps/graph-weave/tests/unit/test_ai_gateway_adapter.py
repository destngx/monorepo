import pytest
from unittest.mock import patch, MagicMock
from src.adapters.ai_gateway_adapter import AIGatewayClient
import httpx

def test_ai_gateway_client_initialization():
    client = AIGatewayClient(base_url="http://test-gateway/v1")
    assert client.base_url == "http://test-gateway/v1"

@patch("httpx.Client.post")
def test_chat_completion_success(mock_post):
    # Mock response
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "choices": [{"message": {"role": "assistant", "content": "Hello!"}}],
        "usage": {"total_tokens": 10}
    }
    mock_post.return_value = mock_response
    
    client = AIGatewayClient(base_url="http://test-gateway/v1")
    messages = [{"role": "user", "content": "Hi"}]
    
    result = client.chat_completion(
        messages=messages,
        provider="openai",
        model="gpt-4"
    )
    
    assert result["choices"][0]["message"]["content"] == "Hello!"
    
    # Verify request headers and payload
    args, kwargs = mock_post.call_args
    assert args[0] == "http://test-gateway/v1/chat/completions"
    assert kwargs["headers"]["X-AI-Provider"] == "openai"
    assert kwargs["json"]["model"] == "gpt-4"
    assert kwargs["json"]["messages"] == messages

@patch("httpx.Client.post")
def test_chat_completion_with_tools(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"choices": []}
    mock_post.return_value = mock_response
    
    client = AIGatewayClient(base_url="http://test-gateway/v1")
    tools = [{"type": "function", "function": {"name": "test_tool"}}]
    
    client.chat_completion(
        messages=[],
        provider="github",
        model="claude-3",
        tools=tools
    )
    
    args, kwargs = mock_post.call_args
    assert kwargs["json"]["tools"] == tools
    assert kwargs["headers"]["X-AI-Provider"] == "github"

@patch("httpx.Client.post")
def test_chat_completion_error_handling(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 500
    mock_response.text = "Internal Server Error"
    
    # Mock raise_for_status to actually raise
    def raise_error():
        raise httpx.HTTPStatusError("500 Error", request=MagicMock(), response=mock_response)
    mock_response.raise_for_status = raise_error
    
    mock_post.return_value = mock_response
    
    client = AIGatewayClient(base_url="http://test-gateway/v1")
    
    with pytest.raises(httpx.HTTPStatusError):
        client.chat_completion(messages=[], provider="test", model="test")
