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
import json
import hashlib
import threading
from ..app_logging import get_logger
from functools import wraps

logger = get_logger(__name__)
from .ai_gateway_adapter import AIGatewayClient
from .mcp import MockMCPServer


# Provider routing (GitHub Copilot, OpenAI) is now handled by AIProviderFactory



VALID_TOOLS = {"load_skill", "search", "verify", "bash", "fs"}


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
    ):
        from .bash_tool import BashTool
        self.mcp_server = mcp_server or MockMCPServer()
        
        # Initialize BashTool with configurable allowed paths
        import os
        workspace_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
        
        # Support adding extra paths via environment variable (comma-separated)
        extra_paths = os.getenv("BASH_TOOL_ALLOWED_PATHS", "").split(",")
        allowed_paths = [workspace_root] + [p.strip() for p in extra_paths if p.strip()]
        
        self.bash_tool = BashTool(allowed_paths=allowed_paths)
        
        from .fs_tool import FileSystemTool
        fs_allowed_paths = os.getenv("FS_TOOL_ALLOWED_PATHS", workspace_root).split(",")
        fs_allowed_paths = [p.strip() for p in fs_allowed_paths if p.strip()]
        fs_trash_path = os.getenv("FS_TOOL_TRASH_PATH")
        self.fs_tool = FileSystemTool(allowed_paths=fs_allowed_paths, trash_path=fs_trash_path)



    def get_tool_definitions(
        self, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """
        Retrieves OpenAI-compatible tool definitions for the registered MCP tools.
        These definitions include function names, descriptions, and parameter schemas,
        which are sent to the LLM to enable tool calling.

        Args:
            allowed_tools: Optional list of tools to include. If None, all tools 
                         registered in the MCP server are returned.

        Returns:
            List of tool definitions in the 'function' format.
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
            elif name == "bash":
                return self.bash(arguments.get("command", ""), arguments.get("cwd"))
            elif name == "fs":
                operation = arguments.pop("operation", "")
                return self.fs(operation, **arguments)

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

    def bash(self, command: str, cwd: Optional[str] = None) -> Dict[str, Any]:
        """Execute a bash command using the BashTool.
        
        Args:
            command: Command string to execute
            cwd: Optional current working directory
            
        Returns:
            Tool response with stdout/stderr or error
            
        Raises:
            ToolExecutionError: If execution fundamentally fails
        """
        try:
            result = self.bash_tool.execute_bash(command, cwd=cwd)
            # Standardize tool response format
            return {
                "tool": "bash",
                "command": command,
                "cwd": result.get("cwd", cwd),
                "status": "success" if result["success"] else "error",
                "exit_code": result.get("exit_code", -1),
                "stdout": result.get("stdout", ""),
                "stderr": result.get("stderr", ""),
                "error": result.get("error", None)
            }
        except Exception as e:
            raise ToolExecutionError(f"Failed to execute bash command '{command}': {str(e)}")

    def fs(self, operation: str, **kwargs) -> Dict[str, Any]:
        """Execute a file system operation using the FileSystemTool.
        
        Args:
            operation: Operation to perform (list_files, read_file, etc.)
            **kwargs: Arguments for the specific operation.
            
        Returns:
            Tool response with result or error.
        """
        try:
            if not hasattr(self.fs_tool, operation):
                return {
                    "tool": "fs",
                    "operation": operation,
                    "status": "error",
                    "error": f"Unknown FS operation: {operation}"
                }
                
            method = getattr(self.fs_tool, operation)
            result = method(**kwargs)
            
            return {
                "tool": "fs",
                "operation": operation,
                "status": "success" if result.get("success", False) else "error",
                **result
            }
        except Exception as e:
            raise ToolExecutionError(f"Failed to execute fs operation '{operation}': {str(e)}")

    def parse_tool_calls(
        self, response_text: str, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """
        Parses potential tool calls from the LLM's raw response text.
        Supports both direct JSON objects and Markdown-wrapped 'tool_call' blocks.

        Args:
            response_text: The raw string response from the LLM.
            allowed_tools: Optional list of allowed tool names to filter for.

        Returns:
            A list of parsed tool call dictionaries, each containing 'tool' and 'arguments'.

        Raises:
            ToolExecutionError: If a found tool call is structurally invalid.
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
