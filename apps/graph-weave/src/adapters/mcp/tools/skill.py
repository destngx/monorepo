from typing import Any, Dict
from ....app_logging import get_logger
from ..infra.exceptions import ToolExecutionError
from ..infra.utils import _tool_response_cache

logger = get_logger(__name__)

@_tool_response_cache
def handle_load_skill(registry: Any, skill_name: str) -> Dict[str, Any]:
    """Load a skill and return its markdown content."""
    try:
        try:
            result = registry.call_tool(
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
