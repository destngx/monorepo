async def orchestrator_node(state: GraphWeaveState, config: GraphWeaveConfig) -> GraphWeaveState:
    """
    Purpose: Generate deterministic routing decision via LLM
    Inputs: state.messages, state.available_skills, state.subagent_summaries
    Outputs: state.routing_directive, state.subagent_payload, state.final_response
    Failure: Retry with backoff (max 3), then FORCE_EXIT
    """
    # Build system prompt with Tier1 skills
    system_prompt = f"""
    You are an orchestrator for workflow {config['workflow_id']}.
    Available skills (Tier1 summaries): {json.dumps(state['available_skills'])}

    You MUST output valid JSON matching this schema:
    {ORCHESTRATOR_OUTPUT_SCHEMA}

    Rules:
    - If you need tool execution, route to SubAgent_X with payload
    - If answer is complete, route to FINISH with final_response
    - If safety violation, route to FORCE_EXIT
    """

    # Structured output via OpenAI function calling or Instructor
    response = await llm.chat_completion(
        messages=[{"role": "system", "content": system_prompt}] + state["messages"],
        response_format=OrchestratorOutput
    )

    # Update token usage
    state["token_usage"]["prompt"] = response.usage.prompt_tokens
    state["token_usage"]["completion"] = response.usage.completion_tokens

    # Extract routing directive
    output = response.choices[0].message.parsed
    state["routing_directive"] = output["routing_directive"]
    state["subagent_payload"] = output["subagent_payload"]
    state["final_response"] = output["final_response"]

    # Track for stagnation detection
    state["stagnation_history"].append(json.dumps(output))

    return state

async def orchestrator(state: OrchestratorState, config: RunnableConfig) -> Dict:
    conf = config["configurable"]
    workflow_def = json.loads(await redis_client.get(f"workflow:{conf['workflow_id']}:{conf['workflow_version']}"))

    system_prompt = assemble_system_prompt(
        workflow_def["system_prompt"], state["available_skills"],
        state["active_mcp_contexts"], state["subagent_summaries"]
    )

    response = await llm.with_structured_output(OrchestratorOutput).ainvoke(
        [SystemMessage(content=system_prompt)] + state["messages"]
    )

    return {
        "messages": [AIMessage(content=response.reasoning)],
        "routing_directive": response.routing_directive,
        "current_subagent_target": response.subagent_target,
        "current_subagent_target_task": response.subagent_payload.get("objective", ""),
        "token_usage": {"orchestrator_tokens": count_tokens(response)}
    }
