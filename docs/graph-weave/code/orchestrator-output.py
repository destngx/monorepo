class OrchestratorOutput(TypedDict):
    reasoning: str
    routing_directive: Literal[
        "Agent_order_tools", "Agent_return_tools", "LOAD_SKILL", "FINISH", "FORCE_EXIT"
    ]
    agent_payload: Dict[str, Any]
    final_response: str
