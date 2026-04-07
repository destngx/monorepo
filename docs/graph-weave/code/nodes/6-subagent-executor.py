async def subagent_executor_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    subagent_graph = StateGraph(SubAgentState)
    subagent_graph.add_node("tool_executor", mcp_tool_node)
    subagent_graph.add_node("summarizer", summarizer_node)

    subagent_state = SubAgentState(
        objective=state["agent_payload"]["objective"],
        required_tools=state["agent_payload"]["required_tools"],
        mcp_contexts=state["active_mcp_contexts"],
    )

    try:
        result = await asyncio.wait_for(
            subagent_graph.ainvoke(subagent_state), timeout=30
        )

        summary = {
            "agent_type": state["routing_directive"],
            "objective": state["agent_payload"]["objective"],
            "result": result["summary"],
            "tool_calls_made": result["tool_count"],
            "success": True,
        }

    except Exception as e:
        summary = {
            "agent_type": state["routing_directive"],
            "objective": state["agent_payload"]["objective"],
            "error": str(e),
            "success": False,
        }

    state["agent_summaries"].append(summary)

    return state


async def subagent_node(state: OrchestratorState, config: RunnableConfig) -> Dict:
    target = state["current_agent_target"]
    mcp_context = state["active_mcp_contexts"][target]
    task = state["current_agent_target_task"]

    tool_result = await execute_tool_isolated(task, mcp_context, config)

    summary = await summarizer_llm.ainvoke(
        f"{mcp_context['summarization_instruction']}\nData: {tool_result.raw_json}"
    )

    return {
        "agent_summaries": [summary.content],
        "token_usage": {"agent_tokens": count_tokens(tool_result)},
    }
