class OrchestratorState(TypedDict):
    messages: Annotated[list, add_messages]
    available_skills: list
    active_mcp_contexts: dict[str, str]
    subagent_summaries: Annotated[list[str], operator.add]
    stagnation_history: Annotated[list[str], keep_last_N]
    routing_directive: str
    current_subagent_target: str
    token_usage: Annotated[dict[str, int], sum_reducer]
    remaining_steps: RemainingSteps
    stagnation_detected: bool

#########
from typing import Annotated, List, Dict, Any, Literal
from langgraph.graph.message import add_messages
from typing_extensions import TypedDict

class GraphWeaveState(TypedDict):
    # Message history (LangGraph managed with reducer)
    messages: Annotated[List[Dict[str, Any]], add_messages]

    # Skill management
    available_skills: Dict[str, Any]  # {skill_id: tier1_summary}
    active_mcp_contexts: Dict[str, Any]  # {skill_id: tier2_schema}

    # Subagent isolation
    subagent_summaries: List[Dict[str, Any]]  # Only summarized results

    # Orchestration
    routing_directive: Literal["SubAgent_order_tools", "SubAgent_return_tools", "FINISH", "FORCE_EXIT"]
    subagent_payload: Dict[str, Any]  # {"objective": "...", "required_tools": [...]}
    final_response: str

    # Stagnation detection
    stagnation_history: Annotated[List[str], lambda x, y: (x + y)[-3:]]  # Last 3 orchestrator outputs

    # Telemetry
    token_usage: Annotated[Dict[str, int], lambda x, y: {**x, **y}]  # {"prompt": 500, "completion": 200}

    # Runtime (managed by LangGraph, DO NOT set manually)
    remaining_steps: int  # LangGraph injects, decrements automatically
