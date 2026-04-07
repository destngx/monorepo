async def skill_loader_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    required_tools = state.get("agent_payload", {}).get("required_tools", [])

    for tool_id in required_tools:
        tier2_key = f"graphweave:skills:{config['tenant_id']}:{tool_id}:tier2"
        tier2_schema = await redis.get(tier2_key)

        if tier2_schema:
            state["active_mcp_contexts"][tool_id] = tier2_schema
        else:
            logger.warning(f"Tier2 schema not found for {tool_id}")
            state["active_mcp_contexts"][tool_id] = {"error": "schema_missing"}

    return state


async def skill_loader(state: OrchestratorState, config: RunnableConfig) -> Dict:
    conf = config["configurable"]
    target_skill = state["routing_directive"].split(":")[1]

    if target_skill in state["active_mcp_contexts"]:
        return {"routing_directive": "IDLE"}

    full_schema = await redis_client.get(
        f"skills:tier2:{target_skill}:{conf['tenant_id']}"
    )
    return {
        "active_mcp_contexts": {target_skill: json.loads(full_schema)},
        "routing_directive": "IDLE",
    }
