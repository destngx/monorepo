import pytest
from unittest.mock import Mock, patch
from src.adapters.ai_provider import AIProviderFactory, LLMClient
from src.adapters.ai_gateway_adapter import AIGatewayClient

class TestAIProviderFactory:
    """Test AIProviderFactory functionality."""

    def test_get_provider_client_returns_unified_instance(self):
        factory = AIProviderFactory()
        client1 = factory.get_provider_client("github-copilot", "gpt-4.1")
        client2 = factory.get_provider_client("openai", "gpt-4o")
        
        # Should return the same AIGatewayClient instance (unified gateway)
        assert client1 is client2
        assert isinstance(client1, AIGatewayClient)

    def test_get_provider_client_caches_instances(self):
        factory = AIProviderFactory()
        client1 = factory.get_provider_client("github-copilot")
        client2 = factory.get_provider_client("github-copilot")
        assert client1 is client2

    def test_factory_uses_provided_client(self):
        mock_client = Mock(spec=LLMClient)
        factory = AIProviderFactory(ai_gateway_client=mock_client)
        
        client = factory.get_provider_client("any-provider")
        assert client is mock_client
