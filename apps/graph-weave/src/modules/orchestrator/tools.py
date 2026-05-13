import json
import logging
from typing import Any, Dict, List, Optional, Callable
from src.models import TraceEntry

logger = logging.getLogger(__name__)

def get_tools(mcp_router: Any, allowed_skills: List[str]) -> List[Dict[str, Any]]:
    """Fetch OpenAI-compatible tool definitions from MCPRouter."""
    try:
        return mcp_router.get_tool_definitions(allowed_skills)
    except Exception as exc:
        logger.warning(f"Could not fetch tool definitions: {exc}")
        return []

def execute_tool_calls(
    mcp_router: Any,
    tool_calls: List[Dict[str, Any]],
    iteration: int,
    node_id: str,
    emit: Callable[[str, Dict[str, Any]], None],
    trace: List[Dict[str, Any]],
    messages: List[Dict[str, Any]]
) -> None:
    for tc in tool_calls:
        tool_name = tc["function"]["name"]
        try:
            tool_args = json.loads(tc["function"]["arguments"])
        except (json.JSONDecodeError, KeyError):
            tool_args = {}
        tool_call_id = tc.get("id", f"call_{tool_name}_{iteration}")

        # Emit streaming event
        emit(
            "orchestrator.tool_called",
            {
                "node_id": node_id,
                "iteration": iteration,
                "tool_name": tool_name,
                "args": tool_args,
            },
        )

        # Record action in trace
        trace.append(
            TraceEntry(
                type="action",
                iteration=iteration,
                content=tool_args,
                tool_name=tool_name,
                tool_args=tool_args,
            ).model_dump()
        )

        # Execute tool
        try:
            tool_result = mcp_router.execute_tool(tool_name, tool_args)
        except Exception as exc:
            logger.warning(f"Orchestrator tool '{tool_name}' failed: {exc}")
            tool_result = {"error": str(exc)}

        # Emit streaming event with result
        emit(
            "orchestrator.tool_called",
            {
                "node_id": node_id,
                "iteration": iteration,
                "tool_name": tool_name,
                "args": tool_args,
                "result": tool_result,
            },
        )

        # Append observation to history
        messages.append(
            {
                "role": "tool",
                "tool_call_id": tool_call_id,
                "content": json.dumps(tool_result),
            }
        )

        # Record observation in trace
        trace.append(
            TraceEntry(
                type="observation",
                iteration=iteration,
                content=tool_result,
                tool_name=tool_name,
            ).model_dump()
        )
