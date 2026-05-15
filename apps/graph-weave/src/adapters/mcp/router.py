import os
import json
from typing import Any, Dict, List, Optional
from ...app_logging import get_logger
from .infra.constants import VALID_TOOLS
from .infra.exceptions import ToolExecutionError
from .infra.utils import format_prompt, filter_allowed_tools
from .infra.parsing import parse_tool_calls, validate_tool_call
from .tools import (
    handle_load_skill,
    handle_search,
    handle_verify,
    handle_bash,
    handle_fs,
    handle_fetch,
    handle_node_registry
)
from .tools.node_registry import NodeRegistryTool
from .registry import ToolRegistry

logger = get_logger(__name__)

class MCPRouter:
    """Routes MCP tool calls and manages provider selection."""

    def __init__(
        self,
        registry: Optional[ToolRegistry] = None,
        node_store: Optional[Any] = None,
    ):
        self.registry = registry or ToolRegistry()
        self.node_store = node_store
        
        # Workspace root for tools
        workspace_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
        
        # BashTool initialization
        from .tools.bash import BashTool
        extra_paths = os.getenv("BASH_TOOL_ALLOWED_PATHS", "").split(",")
        allowed_paths = [workspace_root] + [p.strip() for p in extra_paths if p.strip()]
        self.bash_tool = BashTool(allowed_paths=allowed_paths)
        
        # FileSystemTool initialization
        from .tools.fs import FileSystemTool
        fs_allowed_paths = os.getenv("FS_TOOL_ALLOWED_PATHS", workspace_root).split(",")
        fs_allowed_paths = [os.path.abspath(p.strip()) for p in fs_allowed_paths if p.strip()]
        fs_trash_path = os.getenv(
            "FS_TOOL_TRASH_PATH",
            os.path.join(workspace_root, ".trash"),
        )
        self.fs_tool = FileSystemTool(allowed_paths=fs_allowed_paths, trash_path=fs_trash_path)
        
        # WebTool initialization
        from .tools.web import WebTool
        self.web_tool = WebTool()

        # NodeRegistryTool initialization
        self.node_registry_tool = None
        if self.node_store:
            self.node_registry_tool = NodeRegistryTool(self.node_store)

    def get_tool_definitions(
        self, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        """Retrieves OpenAI-compatible tool definitions."""
        tools = []
        for tool in self.registry.list_tools():
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
        """Execute a tool by name and arguments."""
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
                # Copy arguments to avoid mutation issues if needed, though arguments.pop is used in original
                args = dict(arguments)
                operation = args.pop("operation", "")
                return self.fs(operation, **args)
            elif name == "fetch":
                return self.fetch(arguments.get("url", ""), arguments.get("method", "GET"), arguments.get("headers"))
            elif name == "node_registry":
                args = dict(arguments)
                operation = args.pop("operation", "")
                return self.node_registry(operation, **args)

            # Fallback to direct registry call
            return self.registry.call_tool(name, arguments)

        except Exception as e:
            logger.error(f"Error executing tool {name}: {e}")
            if isinstance(e, ToolExecutionError):
                raise
            raise ToolExecutionError(f"Tool {name} failed: {e}")

    def load_skill(self, skill_name: str) -> Dict[str, Any]:
        return handle_load_skill(self.registry, skill_name)

    def search(self, query: str) -> Dict[str, Any]:
        return handle_search(self.registry, query)

    def verify(self, claim: str) -> Dict[str, Any]:
        return handle_verify(claim)

    def bash(self, command: str, cwd: Optional[str] = None) -> Dict[str, Any]:
        return handle_bash(self.bash_tool, command, cwd)

    def fs(self, operation: str, **kwargs) -> Dict[str, Any]:
        return handle_fs(self.fs_tool, operation, **kwargs)

    def fetch(self, url: str, method: str = "GET", headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
        return handle_fetch(self.web_tool, url, method, headers)

    def node_registry(self, operation: str, **kwargs) -> Dict[str, Any]:
        if not self.node_registry_tool:
             raise ToolExecutionError("Node registry tool is not initialized (no node_store provided)")
        return handle_node_registry(self.node_registry_tool, operation, **kwargs)

    def parse_tool_calls(
        self, response_text: str, allowed_tools: Optional[List[str]] = None
    ) -> List[Dict[str, Any]]:
        return parse_tool_calls(response_text, allowed_tools)

    def _validate_tool_call(
        self, tool_call: Dict[str, Any], allowed_tools: Optional[List[str]] = None
    ) -> bool:
        return validate_tool_call(tool_call, allowed_tools)

    def format_prompt(self, template: str, context: Dict[str, Any]) -> str:
        return format_prompt(template, context)

    def filter_allowed_tools(
        self, all_tools: List[str], allowed_tools: Optional[List[str]] = None
    ) -> List[str]:
        return filter_allowed_tools(all_tools, allowed_tools)
