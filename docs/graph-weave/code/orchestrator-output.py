class OrchestratorOutput(TypedDict):
    reasoning: str
    routing_directive: Literal[
        "SubAgent_order_tools", "SubAgent_return_tools", "FINISH", "FORCE_EXIT"
    ]
    subagent_payload: Dict[str, Any]
    final_response: str
