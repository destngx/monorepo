async def input_guardrail_node(state: GraphWeaveState, config: GraphWeaveConfig) -> GraphWeaveState:
    """
    Purpose: Validate user input before any processing
    Inputs: state.messages[-1]["content"], config.tenant_id
    Outputs: state (unchanged) or raises GuardrailViolation
    Failure: FORCE_EXIT with sanitized error message
    """
    user_input = state["messages"][-1]["content"]

    # Check 1: Length
    if len(user_input) > 10000:
        raise GuardrailViolation("Input exceeds 10K character limit")

    # Check 2: SQL injection patterns
    sql_patterns = ["--", ";", "DROP TABLE", "INSERT INTO"]
    if any(pattern in user_input.upper() for pattern in sql_patterns):
        raise GuardrailViolation("Potentially malicious SQL pattern detected")

    # Check 3: PII leakage (credit card, SSN)
    if re.search(r'\b\d{4}-\d{4}-\d{4}-\d{4}\b', user_input):
        raise GuardrailViolation("Credit card numbers not allowed in input")

    return state  # No mutation

# Node 1: InputGuardrailNode (FastAPI Middleware - Out of Graph)
#
#     Purpose: Validate incoming schema, check quotas, block injections.
async def input_guardrail_middleware(request: ExecutionRequest):
    validated = WorkflowExecutionSchema.parse_obj(request)
    if contains_injection_pattern(validated.user_message):
        raise HTTPException(400, "Input policy violation")

    workflow_def = await redis_client.get(f"workflow:{request.tenant_id}:{request.workflow_id}:{request.workflow_version}")
    if not workflow_def:
        workflow_def = await db.fetch_workflow(...)
        await redis_client.setex(..., 3600, workflow_def)

    if await quota_exceeded(request.tenant_id):
        raise HTTPException(429, "Quota Exceeded")
    return validated, workflow_def

