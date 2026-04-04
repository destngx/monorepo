async def stagnation_detector_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    if len(state["stagnation_history"]) < 3:
        return state

    last_three = state["stagnation_history"][-3:]
    if last_three[0] == last_three[1] == last_three[2]:
        raise StagnationError(
            f"Identical routing decision repeated 3 times: {last_three[0]}",
            partial_results=state.get("subagent_summaries", []),
        )

    return state


async def stagnation_detector(state: OrchestratorState, config: RunnableConfig) -> Dict:
    conf = config["configurable"]
    workflow_def = json.loads(
        await redis_client.get(
            f"workflow:{conf['workflow_id']}:{conf['workflow_version']}"
        )
    )
    window = workflow_def["limits"]["stagnation_window"]
    threshold = workflow_def["limits"]["stagnation_threshold"]

    current_intent = f"{state['routing_directive']}:{state['current_subagent_target']}"
    updated_history = state["stagnation_history"] + [current_intent]
    repeat_count = updated_history[-window:].count(current_intent)

    if repeat_count >= threshold:
        notice = SystemMessage(
            content=f"STAGNATION DETECTED: Aborted after {repeat_count} identical requests."
        )
        return {
            "stagnation_history": [current_intent],
            "stagnation_detected": True,
            "messages": [notice],
            "routing_directive": "FORCE_EXIT",
        }

    return {"stagnation_history": [current_intent], "stagnation_detected": False}
