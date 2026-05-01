from typing import Any, Dict, List, Optional, Protocol, cast
import threading
from .ai_gateway_adapter import AIGatewayClient
from ..app_logging import get_logger

logger = get_logger(__name__)

class LLMClient(Protocol):
    """Protocol for LLM provider clients."""

    def chat_completion(
        self,
        messages: List[Dict[str, str]],
        provider: str,
        model: str,
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.7,
        max_tokens: int = 2000,
    ) -> Dict[str, Any]: ...




class AIProviderFactory:
    """
    Manages the lifecycle and caching of AI Provider clients.
    Separates LLM routing from tool execution.
    """

    def __init__(self, ai_gateway_client: Optional[LLMClient] = None):
        self.ai_gateway_client = ai_gateway_client
        self._provider_cache: Dict[str, LLMClient] = {}
        self._cache_lock = threading.Lock()

    def get_provider_client(
        self,
        provider_name: str,
        model_name: Optional[str] = None,
    ) -> LLMClient:
        """
        Get or create provider client for given provider and model.
        
        Args:
            provider_name: Provider name (github-copilot, openai, etc.)
            model_name: Optional model name override
            
        Returns:
            LLM client instance (AI Gateway)
        """
        if self.ai_gateway_client:
            return self.ai_gateway_client

        cache_key = "ai-gateway-default"
        with self._cache_lock:
            if cache_key in self._provider_cache:
                return self._provider_cache[cache_key]

            logger.info("Initializing AIGatewayClient for AI provider routing")
            client = AIGatewayClient()
            self._provider_cache[cache_key] = cast(LLMClient, client)
            return self._provider_cache[cache_key]
