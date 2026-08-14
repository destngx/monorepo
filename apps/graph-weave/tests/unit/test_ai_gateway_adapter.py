import pytest
from unittest.mock import patch, MagicMock
from src.adapters.ai_gateway_adapter import AIGatewayClient, is_retryable_gateway_error
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
        "usage": {"total_tokens": 10},
    }
    mock_post.return_value = mock_response

    client = AIGatewayClient(base_url="http://test-gateway/v1")
    messages = [{"role": "user", "content": "Hi"}]

    result = client.chat_completion(
        messages=messages, provider="github-copilot", model="gpt-5.4-mini"
    )

    assert result["choices"][0]["message"]["content"] == "Hello!"

    # Verify request headers and payload
    args, kwargs = mock_post.call_args
    assert args[0] == "http://test-gateway/v1/chat/completions"
    assert kwargs["headers"]["X-AI-Provider"] == "github-copilot"
    assert kwargs["json"]["model"] == "gpt-5.4-mini"
    assert kwargs["json"]["messages"] == messages


@patch("httpx.Client.post")
def test_openai_chat_completion_uses_responses_and_normalizes_result(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.text = 'data: {"type":"response.completed","response":{"id":"resp_123","model":"gpt-5.4-mini","output_text":"Hello!","usage":{"input_tokens":4,"output_tokens":6,"total_tokens":10}}}\n\ndata: [DONE]\n'
    mock_post.return_value = mock_response

    client = AIGatewayClient(base_url="http://test-gateway/v1")
    result = client.messages(
        messages=[{"role": "system", "content": "Be helpful."}, {"role": "user", "content": "Hi"}],
        provider="openai",
        model="gpt-5.4-mini",
    )

    args, kwargs = mock_post.call_args
    assert args[0] == "http://test-gateway/v1/responses"
    assert kwargs["json"]["instructions"] == "Be helpful."
    assert kwargs["json"]["input"] == [{"role": "user", "content": "Hi"}]
    assert kwargs["json"]["store"] is False
    assert kwargs["json"]["stream"] is True
    assert result["choices"][0]["message"] == {"role": "assistant", "content": "Hello!"}
    assert result["usage"]["total_tokens"] == 10


@patch("httpx.Client.post")
def test_messages_sends_native_json_schema_format(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.text = 'data: {"type":"response.output_text.delta","delta":"{}"}\n'
    mock_post.return_value = mock_response
    schema = {"type": "object", "properties": {"value": {"type": "string"}}, "required": ["value"]}

    AIGatewayClient(base_url="http://test-gateway/v1").messages(
        messages=[{"role": "user", "content": "Return JSON"}], provider="openai", model="gpt-5.6-luna", output_schema=schema
    )

    assert mock_post.call_args.kwargs["json"]["text"]["format"] == {
        "type": "json_schema", "name": "graph_weave_output", "schema": schema, "strict": True,
    }


@patch("httpx.Client.post")
def test_openai_responses_normalizes_function_calls(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.text = 'data: {"type":"response.completed","response":{"output":[{"type":"function_call","call_id":"call_123","name":"search","arguments":"{\\"query\\":\\"OpenAI\\"}"}]}}\n\ndata: [DONE]\n'
    mock_post.return_value = mock_response

    result = AIGatewayClient(base_url="http://test-gateway/v1").messages(
        messages=[{"role": "user", "content": "Search"}], provider="openai", model="gpt-5.4-mini"
    )

    assert result["choices"][0]["message"]["tool_calls"] == [{
        "id": "call_123", "type": "function", "function": {"name": "search", "arguments": '{"query":"OpenAI"}'},
    }]


@patch("httpx.Client.post")
def test_messages_can_select_chat_completion_handler(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"choices": [{"message": {"role": "assistant", "content": "Legacy"}}]}
    mock_post.return_value = mock_response

    result = AIGatewayClient(base_url="http://test-gateway/v1").messages(
        messages=[], provider="openai", model="gpt-5.4-mini", use_responses=False
    )

    assert mock_post.call_args.args[0] == "http://test-gateway/v1/chat/completions"
    assert result["choices"][0]["message"]["content"] == "Legacy"


@patch("httpx.Client.post")
def test_chat_completion_with_tools(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"choices": []}
    mock_post.return_value = mock_response

    client = AIGatewayClient(base_url="http://test-gateway/v1")
    tools = [{"type": "function", "function": {"name": "test_tool"}}]

    client.chat_completion(
        messages=[], provider="github-copilot", model="claude-3", tools=tools
    )

    args, kwargs = mock_post.call_args
    assert kwargs["json"]["tools"] == tools
    assert kwargs["headers"]["X-AI-Provider"] == "github-copilot"


@patch("httpx.Client.post")
def test_chat_completion_error_handling(mock_post):
    mock_response = MagicMock()
    mock_response.status_code = 500
    mock_response.text = "Internal Server Error"

    # Mock raise_for_status to actually raise
    def raise_error():
        raise httpx.HTTPStatusError(
            "500 Error", request=MagicMock(), response=mock_response
        )

    mock_response.raise_for_status = raise_error

    mock_post.return_value = mock_response

    client = AIGatewayClient(base_url="http://test-gateway/v1")

    with pytest.raises(httpx.HTTPStatusError):
        client.messages(messages=[], provider="test", model="test")


@patch("httpx.Client.post")
def test_chat_completion_retries_transient_gateway_error(mock_post):
    retry_response = MagicMock()
    retry_response.status_code = 502
    retry_response.text = (
        'ERROR HTTP Error rid=632a0e67 status=502 message=Post '
        '"https://token-plan-sgp.xiaomimimo.com/v1/chat/completions": '
        "context deadline exceeded"
    )
    retry_response.raise_for_status.side_effect = httpx.HTTPStatusError(
        "502 Error", request=MagicMock(), response=retry_response
    )

    success_response = MagicMock()
    success_response.status_code = 200
    success_response.json.return_value = {
        "choices": [{"message": {"role": "assistant", "content": "Recovered"}}]
    }

    mock_post.side_effect = [retry_response, success_response]

    client = AIGatewayClient(base_url="http://test-gateway/v1")
    result = client.chat_completion(messages=[], provider="test", model="test")

    assert result["choices"][0]["message"]["content"] == "Recovered"
    assert mock_post.call_count == 2


def test_gateway_502_is_retryable():
    response = MagicMock()
    response.status_code = 502
    exc = httpx.HTTPStatusError("502 Error", request=MagicMock(), response=response)

    assert is_retryable_gateway_error(exc) is True
