def orchestrator_router(state: GraphWeaveState) -> str:
    """Deterministic routing based on orchestrator output"""
    directive = state.get("routing_directive", "")

    if directive.startswith("SubAgent_"):
        return "subagent"
    elif directive == "FINISH":
        return "finish"
    elif directive == "FORCE_EXIT":
        return "force_exit"
    else:
        return "force_exit"  # Unknown directive, safe fallback

def stagnation_router(state: GraphWeaveState) -> str:
    """Route to stagnation handler if needed"""
    if state.get("_stagnation_detected", False):
        return "force_exit"
    return "continue"

def subagent_router(state: GraphWeaveState) -> str:
    """Route to appropriate subagent based on directive"""
    directive = state["routing_directive"]

    # Dynamic mapping from workflow JSON
    subagent_map = {
        "SubAgent_order_tools": "order_subagent",
        "SubAgent_return_tools": "return_subagent",
        # More loaded from workflow config at runtime
    }

    return subagent_map.get(directive, "default_subagent")

def orchestrator_router(state: OrchestratorState, config: RunnableConfig) -> str:
    total_used = sum(state["token_usage"].values())
    if total_used >= config["configurable"]["max_tokens"]:
        return "output_guardrail"

    if state["remaining_steps"] <= 2:
        return "output_guardrail"

    directive = state["routing_directive"]
    if directive.startswith("LOAD_SKILL"): return "skill_loader"
    if directive == "CALL_SUBAGENT": return "stagnation_detector"
    if directive in ["FINISH", "FORCE_EXIT"]: return "output_guardrail"
    return "output_guardrail"

def stagnation_router(state: OrchestratorState) -> str:
    if state["stagnation_detected"]: return "output_guardrail"
    return f"subagent_{state['current_subagent_target']}"

def subagent_router(state: OrchestratorState) -> str:
    return "circuit_breaker"
