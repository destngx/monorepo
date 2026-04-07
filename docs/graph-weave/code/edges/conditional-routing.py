def orchestrator_router(state: GraphWeaveState) -> str:
    directive = state.get("routing_directive", "")

    if directive.startswith("Agent_") or directive.startswith("LOAD_SKILL"):
        return "agent_node"
    elif directive == "FINISH":
        return "finish"
    elif directive == "FORCE_EXIT":
        return "force_exit"
    else:
        return "force_exit"


def stagnation_router(state: GraphWeaveState) -> str:
    if state.get("_stagnation_detected", False):
        return "force_exit"
    return "continue"


def agent_router(state: GraphWeaveState) -> str:
    directive = state["routing_directive"]

    agent_map = {
        "Agent_order_tools": "order_agent",
        "Agent_return_tools": "return_agent",
    }

    return agent_map.get(directive, "default_agent")


def orchestrator_router(state: OrchestratorState, config: RunnableConfig) -> str:
    total_used = sum(state["token_usage"].values())
    if total_used >= config["configurable"]["max_tokens"]:
        return "output_guardrail"

    if state["remaining_steps"] <= 2:
        return "output_guardrail"

    directive = state["routing_directive"]
    if directive.startswith("LOAD_SKILL"):
        return "skill_loader"
    if directive == "CALL_AGENT":
        return "stagnation_detector"
    if directive in ["FINISH", "FORCE_EXIT"]:
        return "output_guardrail"
    return "output_guardrail"


def stagnation_router(state: OrchestratorState) -> str:
    if state["stagnation_detected"]:
        return "output_guardrail"
    return f"agent_{state['current_agent_target']}"


def agent_terminal_router(state: OrchestratorState) -> str:
    return "circuit_breaker"
