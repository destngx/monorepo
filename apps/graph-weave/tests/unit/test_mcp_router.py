"""
Unit tests for MCP Router and provider routing integration.

Tests cover:
- MCPRouter tool methods (load_skill, search, verify)
- Provider routing (GitHub Copilot, OpenAI)
- Provider API key validation
- Model validation
- Temperature/max_tokens validation
- Tools list filtering
- Tool response formatting
- Prompt formatting with variable interpolation
- Tool call parsing (valid and malformed)
- Error scenarios
"""

import pytest
import os
import json
from unittest.mock import Mock, patch, MagicMock
from src.adapters.mcp_router import (
    MCPRouter,
    ProviderConfigError,
    ToolExecutionError,
    PROVIDER_CONFIGS,
)
from src.adapters.mcp import MockMCPServer


class TestMCPRouterToolMethods:
    """Test MCPRouter basic tool methods."""

    def test_load_skill_returns_markdown_content(self):
        router = MCPRouter()
        result = router.load_skill("research")
        assert result["tool"] == "load_skill"
        assert result["skill_name"] == "research"
        assert result["status"] == "success"
        assert "content" in result

    def test_search_returns_results(self):
        router = MCPRouter()
        result = router.search("quantum computing")
        assert result["tool"] == "search"
        assert result["query"] == "quantum computing"
        assert result["status"] == "success"
        assert "results" in result

    def test_verify_returns_verdict(self):
        router = MCPRouter()
        result = router.verify("Earth is flat")
        assert result["tool"] == "verify"
        assert result["claim"] == "Earth is flat"
        assert result["status"] == "success"
        assert "verdict" in result
        assert "confidence" in result

    def test_load_skill_caching(self):
        router = MCPRouter()
        result1 = router.load_skill("research")
        result2 = router.load_skill("research")
        assert result1 == result2

    def test_search_caching(self):
        router = MCPRouter()
        result1 = router.search("test query")
        result2 = router.search("test query")
        assert result1 == result2

    def test_verify_caching(self):
        router = MCPRouter()
        result1 = router.verify("test claim")
        result2 = router.verify("test claim")
        assert result1 == result2


class TestProviderRouting:
    """Test provider routing functionality."""

    def test_get_provider_client_github_with_token(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            client = router.get_provider_client("github")
            assert client is not None

    def test_get_provider_client_openai_with_token(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk_test_key"}):
            router = MCPRouter()
            client = router.get_provider_client("openai")
            assert client is not None

    def test_get_provider_client_caches_instances(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            client1 = router.get_provider_client("github", "claude-3.5-sonnet")
            client2 = router.get_provider_client("github", "claude-3.5-sonnet")
            assert client1 is client2

    def test_get_provider_client_different_models(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            client1 = router.get_provider_client("github", "claude-3.5-sonnet")
            client2 = router.get_provider_client("github", "claude-3-opus")
            assert client1 is not client2


class TestProviderAPIKeyValidation:
    """Test provider API key validation."""

    def test_missing_github_token_raises_error(self):
        with patch.dict(os.environ, {}, clear=True):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("github")
            assert "GITHUB_TOKEN" in str(exc_info.value)

    def test_missing_openai_api_key_raises_error(self):
        with patch.dict(os.environ, {}, clear=True):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("openai")
            assert "OPENAI_API_KEY" in str(exc_info.value)

    def test_invalid_provider_name_raises_error(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("invalid_provider")
            assert "Unknown provider" in str(exc_info.value)


class TestModelValidation:
    """Test model validation for providers."""

    def test_unknown_model_for_github_raises_error(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("github", "unknown-model")
            assert "Invalid model" in str(exc_info.value)

    def test_unknown_model_for_openai_raises_error(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk_test_key"}):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("openai", "unknown-model")
            assert "Invalid model" in str(exc_info.value)

    def test_valid_models_for_github(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            for model in ["claude-3.5-sonnet", "claude-3-opus", "gpt-4"]:
                client = router.get_provider_client("github", model)
                assert client is not None

    def test_valid_models_for_openai(self):
        with patch.dict(os.environ, {"OPENAI_API_KEY": "sk_test_key"}):
            router = MCPRouter()
            for model in ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"]:
                client = router.get_provider_client("openai", model)
                assert client is not None


class TestTemperatureValidation:
    """Test temperature parameter validation."""

    def test_valid_temperature_values(self):
        router = MCPRouter()
        assert router.validate_temperature(0.0) is True
        assert router.validate_temperature(0.5) is True
        assert router.validate_temperature(1.0) is True

    def test_temperature_below_zero_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_temperature(-0.1)
        assert "Temperature must be in [0, 1]" in str(exc_info.value)

    def test_temperature_above_one_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_temperature(1.1)
        assert "Temperature must be in [0, 1]" in str(exc_info.value)

    def test_temperature_non_numeric_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_temperature("high")
        assert "Temperature must be numeric" in str(exc_info.value)


class TestMaxTokensValidation:
    """Test max_tokens parameter validation."""

    def test_valid_max_tokens(self):
        router = MCPRouter()
        assert router.validate_max_tokens(1) is True
        assert router.validate_max_tokens(1000) is True
        assert router.validate_max_tokens(4096) is True

    def test_max_tokens_zero_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_max_tokens(0)
        assert "max_tokens must be >= 1" in str(exc_info.value)

    def test_max_tokens_negative_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_max_tokens(-1)
        assert "max_tokens must be >= 1" in str(exc_info.value)

    def test_max_tokens_non_integer_raises_error(self):
        router = MCPRouter()
        with pytest.raises(ProviderConfigError) as exc_info:
            router.validate_max_tokens(100.5)
        assert "max_tokens must be integer" in str(exc_info.value)


class TestToolsListFiltering:
    """Test tool filtering by allowed list."""

    def test_filter_allowed_tools_no_restriction(self):
        router = MCPRouter()
        all_tools = ["load_skill", "search", "verify"]
        result = router.filter_allowed_tools(all_tools)
        assert result == all_tools

    def test_filter_allowed_tools_with_restriction(self):
        router = MCPRouter()
        all_tools = ["load_skill", "search", "verify"]
        result = router.filter_allowed_tools(all_tools, ["load_skill", "search"])
        assert result == ["load_skill", "search"]

    def test_filter_allowed_tools_empty_restriction(self):
        router = MCPRouter()
        all_tools = ["load_skill", "search", "verify"]
        result = router.filter_allowed_tools(all_tools, [])
        assert result == []

    def test_filter_allowed_tools_subset(self):
        router = MCPRouter()
        all_tools = ["load_skill", "search", "verify"]
        result = router.filter_allowed_tools(all_tools, ["verify"])
        assert result == ["verify"]


class TestPromptFormatting:
    """Test prompt formatting with variable interpolation."""

    def test_format_prompt_single_variable(self):
        router = MCPRouter()
        template = "What is the capital of {country}?"
        context = {"country": "France"}
        result = router.format_prompt(template, context)
        assert result == "What is the capital of France?"

    def test_format_prompt_multiple_variables(self):
        router = MCPRouter()
        template = "{action} the {object} in {location}"
        context = {"action": "Find", "object": "book", "location": "library"}
        result = router.format_prompt(template, context)
        assert result == "Find the book in library"

    def test_format_prompt_no_variables(self):
        router = MCPRouter()
        template = "Hello world"
        context = {}
        result = router.format_prompt(template, context)
        assert result == "Hello world"

    def test_format_prompt_repeated_variable(self):
        router = MCPRouter()
        template = "{word} {word} {word}"
        context = {"word": "hello"}
        result = router.format_prompt(template, context)
        assert result == "hello hello hello"

    def test_format_prompt_with_numbers(self):
        router = MCPRouter()
        template = "Count to {number} and add {increment}"
        context = {"number": 10, "increment": 5}
        result = router.format_prompt(template, context)
        assert result == "Count to 10 and add 5"


class TestToolCallParsing:
    """Test tool call parsing from LLM responses."""

    def test_parse_valid_json_single_tool_call(self):
        router = MCPRouter()
        response = '{"tool": "search", "arguments": {"query": "quantum computing"}}'
        result = router.parse_tool_calls(response)
        assert len(result) == 1
        assert result[0]["tool"] == "search"
        assert result[0]["arguments"]["query"] == "quantum computing"

    def test_parse_valid_json_multiple_tool_calls(self):
        router = MCPRouter()
        response = """[
            {"tool": "search", "arguments": {"query": "AI"}},
            {"tool": "verify", "arguments": {"claim": "AI is intelligent"}}
        ]"""
        result = router.parse_tool_calls(response)
        assert len(result) == 2
        assert result[0]["tool"] == "search"
        assert result[1]["tool"] == "verify"

    def test_parse_tool_calls_with_allowed_tools_filter(self):
        router = MCPRouter()
        response = """[
            {"tool": "search", "arguments": {"query": "test"}},
            {"tool": "verify", "arguments": {"claim": "test"}}
        ]"""
        result = router.parse_tool_calls(response, ["search"])
        assert len(result) == 1
        assert result[0]["tool"] == "search"

    def test_parse_malformed_json_returns_empty(self):
        router = MCPRouter()
        response = "Not valid JSON at all"
        result = router.parse_tool_calls(response)
        assert result == []

    def test_parse_markdown_code_block_tool_call(self):
        router = MCPRouter()
        response = """
        Here's the tool call:
        ```tool_call
        {"tool": "search", "arguments": {"query": "python"}}
        ```
        """
        result = router.parse_tool_calls(response)
        assert len(result) == 1
        assert result[0]["tool"] == "search"

    def test_parse_tool_calls_invalid_tool_name_ignored(self):
        router = MCPRouter()
        response = '{"tool": "invalid_tool", "arguments": {"param": "value"}}'
        result = router.parse_tool_calls(response)
        assert result == []

    def test_parse_tool_calls_missing_arguments_ignored(self):
        router = MCPRouter()
        response = '{"tool": "search"}'
        result = router.parse_tool_calls(response)
        assert result == []

    def test_parse_tool_calls_non_dict_arguments_ignored(self):
        router = MCPRouter()
        response = '{"tool": "search", "arguments": "not a dict"}'
        result = router.parse_tool_calls(response)
        assert result == []


class TestToolResponseFormatting:
    """Test tool response formatting."""

    def test_load_skill_response_format(self):
        router = MCPRouter()
        result = router.load_skill("test_skill")
        assert "tool" in result
        assert "skill_name" in result
        assert "status" in result
        assert "content" in result

    def test_search_response_format(self):
        router = MCPRouter()
        result = router.search("test query")
        assert "tool" in result
        assert "query" in result
        assert "status" in result
        assert "results" in result

    def test_verify_response_format(self):
        router = MCPRouter()
        result = router.verify("test claim")
        assert "tool" in result
        assert "claim" in result
        assert "status" in result
        assert "verdict" in result
        assert "confidence" in result


class TestErrorScenarios:
    """Test error handling in various scenarios."""

    def test_invalid_provider_name_in_error_message(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("nonexistent")
            error_msg = str(exc_info.value)
            assert "nonexistent" in error_msg
            assert "github" in error_msg or "openai" in error_msg

    def test_invalid_model_in_error_message(self):
        with patch.dict(os.environ, {"GITHUB_TOKEN": "gho_test_token"}):
            router = MCPRouter()
            with pytest.raises(ProviderConfigError) as exc_info:
                router.get_provider_client("github", "bad-model")
            error_msg = str(exc_info.value)
            assert "bad-model" in error_msg
            assert "github" in error_msg

    def test_tool_call_validation_non_dict(self):
        router = MCPRouter()
        result = router.parse_tool_calls('["not", "valid"]')
        assert result == []

    def test_parse_tool_calls_mixed_valid_invalid(self):
        router = MCPRouter()
        response = """[
            {"tool": "search", "arguments": {"query": "test"}},
            {"tool": "invalid", "arguments": {"param": "value"}},
            {"tool": "verify", "arguments": {"claim": "test"}}
        ]"""
        result = router.parse_tool_calls(response)
        assert len(result) == 2
        tools = [r["tool"] for r in result]
        assert "search" in tools
        assert "verify" in tools
        assert "invalid" not in tools
