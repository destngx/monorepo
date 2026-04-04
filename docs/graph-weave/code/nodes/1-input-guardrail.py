async def input_guardrail_node(
    state: GraphWeaveState, config: GraphWeaveConfig
) -> GraphWeaveState:
    user_input = state["messages"][-1]["content"]

    if len(user_input) > 10000:
        raise GuardrailViolation("Input exceeds 10K character limit")

    sql_patterns = ["--", ";", "DROP TABLE", "INSERT INTO"]
    if any(pattern in user_input.upper() for pattern in sql_patterns):
        raise GuardrailViolation("Potentially malicious SQL pattern detected")

    if re.search(r"\b\d{4}-\d{4}-\d{4}-\d{4}\b", user_input):
        raise GuardrailViolation("Credit card numbers not allowed in input")

    return state


async def input_guardrail_middleware(request: ExecutionRequest):
    validated = WorkflowExecutionSchema.parse_obj(request)
    if contains_injection_pattern(validated.user_message):
        raise HTTPException(400, "Input policy violation")

    workflow_def = await redis_client.get(
        f"workflow:{request.tenant_id}:{request.workflow_id}:{request.workflow_version}"
    )
    if not workflow_def:
        workflow_def = await db.fetch_workflow(...)
        await redis_client.setex(..., 3600, workflow_def)

    if await quota_exceeded(request.tenant_id):
        raise HTTPException(429, "Quota Exceeded")
    return validated, workflow_def
