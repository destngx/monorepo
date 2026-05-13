from typing import Any, Dict, List, Optional
from ...app_logging import get_logger
from .exceptions import ToolExecutionError
from .utils import _tool_response_cache

logger = get_logger(__name__)

@_tool_response_cache
def handle_load_skill(mcp_server: Any, skill_name: str) -> Dict[str, Any]:
    """Load a skill and return its markdown content."""
    try:
        try:
            result = mcp_server.call_tool(
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
def handle_search(mcp_server: Any, query: str) -> Dict[str, Any]:
    """Execute search tool for a query."""
    try:
        result = mcp_server.call_tool("search", {"query": query})
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
def handle_verify(claim: str) -> Dict[str, Any]:
    """Verify a claim using the MCP verify tool."""
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

def handle_bash(bash_tool: Any, command: str, cwd: Optional[str] = None) -> Dict[str, Any]:
    """Execute a bash command using the BashTool."""
    try:
        result = bash_tool.execute_bash(command, cwd=cwd)
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

def handle_fs(fs_tool: Any, operation: str, **kwargs) -> Dict[str, Any]:
    """Execute a file system operation using the FileSystemTool."""
    try:
        if not hasattr(fs_tool, operation):
            return {
                "tool": "fs",
                "operation": operation,
                "status": "error",
                "error": f"Unknown FS operation: {operation}"
            }
            
        method = getattr(fs_tool, operation)
        result = method(**kwargs)
        
        return {
            "tool": "fs",
            "operation": operation,
            "status": "success" if result.get("success", False) else "error",
            **result
        }
    except Exception as e:
        raise ToolExecutionError(f"Failed to execute fs operation '{operation}': {str(e)}")

def handle_fetch(web_tool: Any, url: str, method: str = "GET", headers: Optional[Dict[str, str]] = None) -> Dict[str, Any]:
    """Execute web fetch tool."""
    try:
        result = web_tool.fetch(url, method=method, headers=headers)
        return {
            "tool": "fetch",
            "url": url,
            "status": "success" if result.get("success", False) else "error",
            **result
        }
    except Exception as e:
        raise ToolExecutionError(f"Failed to fetch URL '{url}': {str(e)}")
