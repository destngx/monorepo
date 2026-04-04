from langgraph.graph import StateGraph, END
from langgraph.checkpoint.redis import AsyncRedisSaver

# Create base graph
builder = StateGraph(GraphWeaveState)

# Add nodes
builder.add_node("input_guardrail", input_guardrail_node)
builder.add_node("graph_initializer", graph_initializer_node)
builder.add_node("skill_loader", skill_loader_node)
builder.add_node("orchestrator", orchestrator_node)
builder.add_node("stagnation_detector", stagnation_detector_node)
builder.add_node("subagent_executor", subagent_executor_node)
builder.add_node("circuit_breaker", circuit_breaker_watchdog_node)
builder.add_node("output_guardrail", output_guardrail_node)

# Add conditional edges
builder.add_conditional_edges("input_guardrail", input_guardrail_router)
builder.add_edge("graph_initializer", "skill_loader")
builder.add_edge("skill_loader", "circuit_breaker")
builder.add_edge("circuit_breaker", "orchestrator")
builder.add_edge("orchestrator", "stagnation_detector")
builder.add_conditional_edges("stagnation_detector", stagnation_router)
builder.add_conditional_edges("orchestrator", orchestrator_router, {
    "subagent": "subagent_executor",
    "finish": "output_guardrail",
    "force_exit": END
})
builder.add_edge("subagent_executor", "circuit_breaker")  # Loop back
builder.add_edge("output_guardrail", END)

# Compile with Redis checkpointer
checkpointer = AsyncRedisSaver(redis_connection)
graph = builder.compile(
    checkpointer=checkpointer,
    interrupt_before=["circuit_breaker"]  # Allow kill switch injection
)

# Dynamic node injection (from workflow JSON)
def inject_custom_nodes(workflow_json: dict, builder: StateGraph):
    """Inject tenant-specific nodes from workflow definition"""
    for node_name, node_config in workflow_json.get("custom_nodes", {}).items():
        builder.add_node(
            node_name,
            create_dynamic_node(node_config)  # Factory pattern
        )
    return builder

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

    subagent_map = {f"subagent_{sa['name']}": f"subagent_{sa['name']}" for sa in workflow_def["subagents"]}
    subagent_map["output_guardrail"] = "output_guardrail"
    builder.add_conditional_edges("stagnation_detector", stagnation_router, subagent_map)

    checkpointer = AsyncRedisSaver.from_conn_string(REDIS_URL)
    return builder.compile(checkpointer=checkpointer)
