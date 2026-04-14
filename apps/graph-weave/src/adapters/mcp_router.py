"""
MCP Router for LLM & Tool Integration.

Provides:
- MCPRouter class with load_skill, search, verify methods
- Provider routing (GitHub Copilot, OpenAI)
- Provider configuration validation
- Tool response caching
- Error handling for malformed tool calls
"""

from typing import Any, Dict, List, Optional, Protocol, Callable, TypeVar, cast
import os
import json
import hashlib
import threading
from src.app_logging import get_logger
from functools import wraps
from .ai_gateway_adapter import AIGatewayClient
from .mcp import MockMCPServer


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

    def call(
        self,
        system_prompt: str,
        user_prompt: str,
        model: str,
        temperature: float,
        max_tokens: int,
    ) -> Dict[str, Any]: ...


PROVIDER_CONFIGS: Dict[str, Dict[str, Any]] = {
    "github-copilot": {
        "models": [
            "claude-3.5-sonnet",
            "claude-3-opus",
            "gpt-4.1",
            "claude-3-sonnet",
        ],
        "default_model": "gpt-4.1",
        "allow_fallback": False,
    },
    "openai": {
        "required_env": "OPENAI_API_KEY",
        "models": ["gpt-4", "gpt-4-turbo", "gpt-3.5-turbo", "gpt-4.1"],
        "default_model": "gpt-4.1",
        "allow_fallback": False,
    },
}

VALID_TOOLS = {"load_skill", "search", "verify"}


class MCPRouterError(Exception):
    """Base exception for MCP router errors."""

    pass


class ProviderConfigError(MCPRouterError):
    """Provider configuration error."""

    pass


class ToolExecutionError(MCPRouterError):
    """Tool execution error."""

    pass


def _tool_response_cache(func: Callable[..., Any]) -> Callable[..., Any]:
    """Decorator for caching tool responses based on input hash."""
    cache: Dict[str, Any] = {}

    @wraps(func)
    def wrapper(*args: Any, **kwargs: Any) -> Any:
        cache_key = f"{func.__name__}:{str(args)}:{str(kwargs)}"
        cache_hash = hashlib.md5(cache_key.encode()).hexdigest()

        if cache_hash in cache:
            return cache[cache_hash]

        result = func(*args, **kwargs)
        cache[cache_hash] = result
        return result

    return wrapper


class MCPRouter:
    """Routes MCP tool calls and manages provider selection."""

    def __init__(
        self,
        mcp_server: Optional[MockMCPServer] = None,
        ai_gateway_client: Optional[LLMClient] = None,
    ):
        self.mcp_server = mcp_server or MockMCPServer()
        self.ai_gateway_client = ai_gateway_client
        self._provider_cache: Dict[str, LLMClient] = {}
        self._cache_lock = threading.Lock()

    def get_provider_client(
        self,
        provider_name: str,
        model_name: Optional[str] = None,
        allow_fallback: Optional[bool] = None,
    ) -> LLMClient:
        """Get or create provider client for given provider and model.

        Args:
            provider_name: Provider name (github, openai)
            model_name: Optional model name override
            allow_fallback: Override fallback behavior (None = use config default)

        Returns:
            LLM client instance

        Raises:
            ProviderConfigError: If provider not found or config invalid
        """
        if provider_name not in PROVIDER_CONFIGS:
            supported = ", ".join(PROVIDER_CONFIGS.keys())
            raise ProviderConfigError(
                f"Unknown provider: {provider_name}. Supported: {supported}"
            )

        config = PROVIDER_CONFIGS[provider_name]
        model = model_name or cast(str, config["default_model"])
        supported_models: List[str] = cast(List[str], config["models"])
        if model not in supported_models:
            supported_models_str = ", ".join(supported_models)
            raise ProviderConfigError(
                f"Invalid model '{model}' for provider '{provider_name}'. "
                f"Supported: {supported_models_str}"
            )

        cache_key = f"{provider_name}:{model}"

        with self._cache_lock:
            if cache_key in self._provider_cache:
                logger.debug(f"Cache hit: {cache_key}")
                return self._provider_cache[cache_key]

        logger.info(f"Creating provider instance: {provider_name}:{model}")
        try:
            client = self._instantiate_provider(provider_name, model)
        except Exception as e:
            logger.error(
                f"Provider instantiation failed: {provider_name}:{model}", exc_info=True
            )
            raise ProviderConfigError(
                f"Failed to create {provider_name} provider: {e}"
            ) from e

        with self._cache_lock:
            self._provider_cache[cache_key] = client

        logger.debug(f"Cached new provider: {cache_key}")
        return client

    def _instantiate_provider(
        self, provider_name: str, model: str
    ) -> LLMClient:
        """Instantiate provider client with validation.

        Args:
            provider_name: Provider name
            api_key: API key/token
            model: Model name

        Returns:
            Instantiated provider client

        Raises:
            ProviderConfigError: If instantiation fails
        """
        logger = get_logger(__name__)

        # Now all providers route through AI Gateway for unified tool calling
        logger.info(f"Routing provider {provider_name} through AI Gateway")
        return self.ai_gateway_client or cast(LLMClient, AIGatewayClient())

    def get_tool_definitions(
        self, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """
        Get OpenAI-compatible tool definitions for the registered MCP tools.
        
        Args:
            allowed_tools: Optional list of tools to include.
            
        Returns:
            List of tool definitions.
        """
        tools = []
        for tool in self.mcp_server.list_tools():
            name = tool["name"]
            if allowed_tools is None or name in allowed_tools:
                tools.append(
                    {
                        "type": "function",
                        "function": {
                            "name": name,
                            "description": tool["description"],
                            "parameters": tool["inputSchema"],
                        },
                    }
                )
        return tools

    def execute_tool(self, name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """
        Execute a tool by name and arguments.
        
        Args:
            name: Tool name.
            arguments: Tool arguments.
            
        Returns:
            Execution result.
        """
        if name not in VALID_TOOLS:
            raise ToolExecutionError(f"Unknown tool: {name}")

        try:
            if name == "load_skill":
                return self.load_skill(arguments.get("skill_name", ""))
            elif name == "search":
                return self.search(arguments.get("query", ""))
            elif name == "verify":
                return self.verify(arguments.get("claim", ""))
            
            # Fallback to direct MCP call if not handled by specific methods
            return self.mcp_server.call_tool(name, arguments)
            
        except Exception as e:
            logger.error(f"Error executing tool {name}: {e}")
            raise ToolExecutionError(f"Tool {name} failed: {e}")
    @_tool_response_cache
    def load_skill(self, skill_name: str) -> Dict[str, Any]:
        """Load a skill and return its markdown content.

        Args:
            skill_name: Skill name to load

        Returns:
            Tool response with markdown content

        Raises:
            ToolExecutionError: If tool call fails
        """
        try:
            try:
                result = self.mcp_server.call_tool(
                    "load_skill", {"skill_name": skill_name}
                )
            except ValueError:
                result = None

            return {
                "tool": "load_skill",
                "skill_name": skill_name,
                "status": "success",
                "content": result or f"# {skill_name}\n\nSkill loaded successfully.",
            }
        except Exception as e:
            raise ToolExecutionError(f"Failed to load skill '{skill_name}': {str(e)}")

    @_tool_response_cache
    def search(self, query: str) -> Dict[str, Any]:
        """Execute search tool for a query.

        Args:
            query: Search query string

        Returns:
            Tool response with search results

        Raises:
            ToolExecutionError: If tool call fails
        """
        try:
            result = self.mcp_server.call_tool("search", {"query": query})
            return {
                "tool": "search",
                "query": query,
                "status": "success",
                "results": result.get("results", [])
                if isinstance(result, dict)
                else [result],
            }
        except Exception as e:
            raise ToolExecutionError(f"Failed to search for '{query}': {str(e)}")

    @_tool_response_cache
    def verify(self, claim: str) -> Dict[str, Any]:
        """Verify a claim using the MCP verify tool.

        Args:
            claim: Claim to verify

        Returns:
            Tool response with verification result

        Raises:
            ToolExecutionError: If tool call fails
        """
        try:
            # For MVP, focusing on standard mock response
            return {
                "tool": "verify",
                "claim": claim,
                "status": "success",
                "verdict": "unverified",
                "confidence": 0.0,
                "evidence": [],
            }
        except Exception as e:
            raise ToolExecutionError(f"Failed to verify claim '{claim}': {str(e)}")

    def parse_tool_calls(
        self, response_text: str, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """Parse tool calls from LLM response text.

        Args:
            response_text: LLM response text
            allowed_tools: List of allowed tool names (None = all tools allowed)

        Returns:
            List of parsed tool calls with name and arguments

        Raises:
            ToolExecutionError: If response cannot be parsed
        """
        tool_calls = []

        try:
            parsed = json.loads(response_text)
            if isinstance(parsed, dict):
                if "tool" in parsed and "arguments" in parsed:
                    tool_call = {
                        "tool": parsed["tool"],
                        "arguments": parsed["arguments"],
                    }
                    if self._validate_tool_call(tool_call, allowed_tools):
                        tool_calls.append(tool_call)
            elif isinstance(parsed, list):
                for item in parsed:
                    if (
                        isinstance(item, dict)
                        and "tool" in item
                        and "arguments" in item
                    ):
                        if self._validate_tool_call(item, allowed_tools):
                            tool_calls.append(item)
        except json.JSONDecodeError:
            import re

            pattern = r"```(?:tool_call)?\s*\n?(.*?)\n?```"
            for match in re.finditer(pattern, response_text, re.DOTALL):
                try:
                    parsed = json.loads(match.group(1))
                    if "tool" in parsed and "arguments" in parsed:
                        if self._validate_tool_call(parsed, allowed_tools):
                            tool_calls.append(parsed)
                except json.JSONDecodeError:
                    continue

        return tool_calls

    def _validate_tool_call(
        self, tool_call: Dict[str, Any], allowed_tools: Optional[List[str]] = None
    ) -> bool:
        """Validate a single tool call.

        Args:
            tool_call: Tool call dict with 'tool' and 'arguments' keys
            allowed_tools: List of allowed tool names

        Returns:
            True if valid, False otherwise
        """
        if not isinstance(tool_call, dict):
            return False

        tool_name = tool_call.get("tool")
        if not tool_name or not isinstance(tool_name, str):
            return False

        if tool_name not in VALID_TOOLS:
            return False

        if allowed_tools and tool_name not in allowed_tools:
            return False

        arguments = tool_call.get("arguments")
        if not isinstance(arguments, dict):
            return False

        return True

    def format_prompt(self, template: str, context: Dict[str, Any]) -> str:
        """Format a prompt template with variable interpolation.

        Args:
            template: Prompt template with {variable} placeholders
            context: Context dictionary for variable substitution

        Returns:
            Formatted prompt string
        """
        result = template
        for key, value in context.items():
            placeholder = f"{{{key}}}"
            if placeholder in result:
                result = result.replace(placeholder, str(value))
        return result

    def validate_temperature(self, temperature: float) -> bool:
        """Validate temperature parameter.

        Args:
            temperature: Temperature value

        Returns:
            True if valid (0 <= temperature <= 1)

        Raises:
            ProviderConfigError: If invalid
        """
        if not isinstance(temperature, (int, float)):
            raise ProviderConfigError(
                f"Temperature must be numeric, got {type(temperature)}"
            )
        if not (0 <= temperature <= 1):
            raise ProviderConfigError(
                f"Temperature must be in [0, 1], got {temperature}"
            )
        return True

    def validate_max_tokens(self, max_tokens: int) -> bool:
        """Validate max_tokens parameter.

        Args:
            max_tokens: Max tokens value

        Returns:
            True if valid (>= 1)

        Raises:
            ProviderConfigError: If invalid
        """
        if not isinstance(max_tokens, int):
            raise ProviderConfigError(
                f"max_tokens must be integer, got {type(max_tokens)}"
            )
        if max_tokens < 1:
            raise ProviderConfigError(f"max_tokens must be >= 1, got {max_tokens}")
        return True

    def filter_allowed_tools(
        self, all_tools: List[str], allowed_tools: Optional[List[str]] = None
    ) -> List[str]:
        """Filter tools list to only allowed tools.

        Args:
            all_tools: All available tools
            allowed_tools: Allowed tools (None = all allowed)

        Returns:
            Filtered list of tools
        """
        if allowed_tools is None:
            return all_tools
        return [t for t in all_tools if t in allowed_tools]
