async def output_guardrail_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    final = state["final_response"]

    if re.search(r"\b\d{3}-\d{2}-\d{4}\b", final):
        raise GuardrailViolation("SSN detected in output")

    profanity_list = await redis.smembers("graphweave:guardrails:profanity")
    if any(word in final.lower() for word in profanity_list):
        state["final_response"] = "[Response redacted due to policy violation]"

    if len(final) > 2000:
        state["final_response"] = final[:1997] + "..."

    return state


async def output_guardrail(state: OrchestratorState, config: RunnableConfig) -> Dict:
    conf = config["configurable"]
    workflow_def = json.loads(
        await redis_client.get(
            f"workflow:{conf['workflow_id']}:{conf['workflow_version']}"
        )
    )
    final_message = state["messages"][-1].content

    if workflow_def["guardrails"]["output"]["pii_detection"] and detect_pii(
        final_message
    ):
        final_message = redact_pii(final_message)

    for topic in workflow_def["guardrails"]["output"]["blocked_topics"]:
        if topic.lower() in final_message.lower():
            final_message = "Response blocked due to restricted topic policy."
            break

    return {"messages": [AIMessage(content=final_message)]}
