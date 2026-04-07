async def graph_initializer_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    workflow_key = f"graphweave:workflow:{config['tenant_id']}:{config['workflow_id']}:v{config['workflow_version']}"
    workflow_json = await redis.get(workflow_key)

    if not workflow_json:
        raise WorkflowNotFoundError(
            f"Workflow {config['workflow_id']} v{config['workflow_version']} not found"
        )

    state["_workflow_limits"] = workflow_json["limits"]
    state["_workflow_skills"] = workflow_json["skills"]

    tier1_skills = {}
    for skill_id in workflow_json["skills"]["always_load_tier1"]:
        tier1_key = f"graphweave:skills:{config['tenant_id']}:{skill_id}:tier1"
        tier1_skills[skill_id] = await redis.get(tier1_key)

    state["available_skills"] = tier1_skills
    return state


async def graph_initializer(state: OrchestratorState, config: RunnableConfig) -> Dict:
    conf = config["configurable"]
    workflow_def = json.loads(
        await redis_client.get(
            f"workflow:{conf['workflow_id']}:{conf['workflow_version']}"
        )
    )
    tier1_summaries = await load_skill_tier1(conf["tenant_id"], workflow_def["skills"])
    return {
        "available_skills": tier1_summaries,
        "active_mcp_contexts": {},
        "agent_summaries": [],
        "stagnation_history": [],
        "routing_directive": "IDLE",
        "stagnation_detected": False,
    }
