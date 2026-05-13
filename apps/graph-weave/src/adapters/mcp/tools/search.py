from typing import Any, Dict
from ....app_logging import get_logger
from ..infra.exceptions import ToolExecutionError
from ..infra.utils import _tool_response_cache

logger = get_logger(__name__)

@_tool_response_cache
def handle_search(registry: Any, query: str) -> Dict[str, Any]:
    """Execute search tool for a query."""
    try:
        result = registry.call_tool("search", {"query": query})
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
