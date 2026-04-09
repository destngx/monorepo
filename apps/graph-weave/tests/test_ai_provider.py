import pytest
import json
from src.adapters.ai_provider import MockAIProvider


class TestMockAIProviderInstantiation:
    def test_instantiation(self):
        provider = MockAIProvider()
        assert provider is not None
        assert provider._call_count == 0

    def test_with_custom_initialization(self):
        provider = MockAIProvider()
        assert hasattr(provider, "call")
        assert hasattr(provider, "reset")


class TestMockAIProviderInterface:
    def test_call_returns_dict(self):
        provider = MockAIProvider()
        result = provider.call(system_prompt="Test system", user_prompt="Test user")
        assert isinstance(result, dict)
        assert "content" in result
        assert "tokens_used" in result
        assert "model" in result
        assert "call_count" in result

    def test_call_with_all_parameters(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="System",
            user_prompt="User",
            model="gpt-4",
            temperature=0.7,
            max_tokens=2000,
        )
        assert result["model"] == "gpt-4"

    def test_content_is_valid_json(self):
        provider = MockAIProvider()
        result = provider.call("test", "test")
        content = result["content"]
        parsed = json.loads(content)
        assert isinstance(parsed, dict)

    def test_call_count_increments(self):
        provider = MockAIProvider()
        provider.call("test", "test")
        assert provider._call_count == 1
        provider.call("test", "test")
        assert provider._call_count == 2


class TestMockAIProviderRouting:
    def test_research_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="You are a research agent searching for information",
            user_prompt="Search for earnings data",
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert "findings" in content or "sources" in content
        assert content["status"] == "research_complete"

    def test_sql_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="You are a SQL agent",
            user_prompt="Query the database for performance metrics",
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert "query" in content
        assert content["status"] == "query_complete"

    def test_synthesis_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="Synthesize the findings",
            user_prompt="Combine research and sql results",
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert "summary" in content
        assert content["status"] == "synthesis_complete"

    def test_classification_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="Classify the request",
            user_prompt="Categorize this as financial or operational",
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert "classification" in content

    def test_stagnation_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="Check for stagnation",
            user_prompt="Detect if we are looping repeatedly",
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert "detected_stagnation" in content

    def test_default_routing(self):
        provider = MockAIProvider()
        result = provider.call(
            system_prompt="Generic prompt", user_prompt="Generic task"
        )
        content = json.loads(result["content"])
        assert "status" in content
        assert content["status"] == "completed"


class TestMockAIProviderDeterminism:
    def test_same_prompt_returns_same_response(self):
        provider = MockAIProvider()
        system = "You are a research agent"
        user = "Search for earnings"

        result1 = provider.call(system, user)
        provider.reset()
        result2 = provider.call(system, user)

        assert result1["content"] == result2["content"]

    def test_different_prompts_return_different_responses(self):
        provider = MockAIProvider()
        result1 = provider.call("Research prompt", "search")
        result2 = provider.call("SQL prompt", "query")

        assert result1["content"] != result2["content"]


class TestMockAIProviderTokenEstimation:
    def test_token_estimation_calculation(self):
        provider = MockAIProvider()
        result = provider.call("test", "test")
        content = result["content"]
        tokens_used = result["tokens_used"]

        expected_tokens = max(1, len(content) // 4)
        assert tokens_used == expected_tokens

    def test_tokens_positive_integer(self):
        provider = MockAIProvider()
        result = provider.call("test", "test")
        assert result["tokens_used"] > 0
        assert isinstance(result["tokens_used"], int)


class TestMockAIProviderReset:
    def test_reset_clears_call_count(self):
        provider = MockAIProvider()
        provider.call("test", "test")
        provider.call("test", "test")
        assert provider._call_count == 2

        provider.reset()
        assert provider._call_count == 0
