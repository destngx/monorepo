from typing import Annotated, Any, Dict, List, Literal
from typing_extensions import TypedDict
from langgraph.graph.message import add_messages


class GraphWeaveState(TypedDict):
    messages: Annotated[List[Dict[str, Any]], add_messages]
    available_skills: Dict[str, Any]
    active_mcp_contexts: Dict[str, Any]
    subagent_summaries: List[Dict[str, Any]]
    routing_directive: Literal[
        "SubAgent_order_tools", "SubAgent_return_tools", "FINISH", "FORCE_EXIT"
    ]
    subagent_payload: Dict[str, Any]
    final_response: str
    stagnation_history: Annotated[List[str], lambda x, y: (x + y)[-3:]]
    token_usage: Annotated[Dict[str, int], lambda x, y: {**x, **y}]
    remaining_steps: int
