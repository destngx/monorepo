class OrchestratorOutput(TypedDict):
    reasoning: str  # Chain-of-thought for audit
    routing_directive: Literal["SubAgent_order_tools", "SubAgent_return_tools", "FINISH", "FORCE_EXIT"]
    subagent_payload: Dict[str, Any]  # Must contain "objective" and "required_tools"
    final_response: str  # Populated only if routing_directive == "FINISH"
