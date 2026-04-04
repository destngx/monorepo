async def circuit_breaker_watchdog_node(state: GraphWeaveState, config: GraphWeaveConfig) -> GraphWeaveState:
    """
    Purpose: Check Redis kill flags before each orchestrator iteration
    Inputs: config.tenant_id, config.workflow_id, config.thread_id
    Outputs: state or raises KillSwitchException
    Failure: Interrupt graph execution immediately
    """
    # Check tenant-level kill switch
    tenant_kill = await redis.get(f"graphweave:circuit_breaker:tenant:{config['tenant_id']}:kill")
    if tenant_kill and tenant_kill == "1":
        raise KillSwitchException(f"Tenant {config['tenant_id']} kill switch active")

    # Check workflow-level kill switch
    workflow_kill = await redis.get(f"graphweave:circuit_breaker:workflow:{config['tenant_id']}:{config['workflow_id']}:kill")
    if workflow_kill and workflow_kill == "1":
        raise KillSwitchException(f"Workflow {config['workflow_id']} kill switch active")

    # Check thread-level kill switch
    thread_kill = await redis.get(f"graphweave:circuit_breaker:thread:{config['thread_id']}:kill")
    if thread_kill and thread_kill == "1":
        raise KillSwitchException(f"Thread {config['thread_id']} kill switch active")

    return state


from langgraph.errors import NodeInterrupt

async def circuit_breaker_watchdog(state: OrchestratorState, config: RunnableConfig) -> Dict:
    thread_id = config["configurable"]["thread_id"]
    kill_signal = await redis_client.get(f"kill:{thread_id}")

    if kill_signal:
        from langgraph.types import interrupt
        interrupt("Thread terminated by administrator kill-switch")

    total_used = sum(state["token_usage"].values())
    if total_used >= config["configurable"]["max_tokens"]:
        return {"routing_directive": "FORCE_EXIT"}

    return {}
