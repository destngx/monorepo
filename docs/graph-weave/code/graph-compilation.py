from langgraph.checkpoint.redis import AsyncRedisSaver
from langgraph.graph import END, StateGraph


async def compile_universal_interpreter_graph(workflow_def: dict) -> CompiledStateGraph:
    builder = StateGraph(OrchestratorState)

    builder.add_node("initializer", graph_initializer)
    builder.add_node("orchestrator", orchestrator)
    builder.add_node("skill_loader", skill_loader)
    builder.add_node("stagnation_detector", stagnation_detector)
    builder.add_node("circuit_breaker", circuit_breaker_watchdog)
    builder.add_node("output_guardrail", output_guardrail)

    for subagent in workflow_def["subagents"]:
        node_name = f"subagent_{subagent['name']}"
        builder.add_node(node_name, create_subagent_node(subagent))
        builder.add_edge(node_name, "circuit_breaker")

    builder.set_entry_point("initializer")
    builder.add_edge("initializer", "orchestrator")
    builder.add_edge("skill_loader", "orchestrator")
    builder.add_edge("circuit_breaker", "orchestrator")
    builder.add_edge("output_guardrail", END)
    builder.add_conditional_edges("orchestrator", orchestrator_router)
    builder.add_conditional_edges("stagnation_detector", stagnation_router)

    checkpointer = AsyncRedisSaver.from_conn_string(REDIS_URL)
    return builder.compile(checkpointer=checkpointer)
