# GW-MVP-E2E-002: End-to-End LangGraph Agent Execution via AI Gateway

**Objective**: Verify real LangGraph agent execution calling real MCP tools (load_skill, search, verify) through the `ai-gateway` proxy. This proves the integration of the unified AI proxy layer and its impact on the tool calling loop.

**Phase**: [MVP]

## Test Scenario

**Given**:

- Real LangGraph executor integrated with `AIGatewayClient`.
- `ai-gateway` application running on `port 8080`.
- Real MCP server providing: `load_skill`, `search`, `verify` tools.
- Valid `AI_GATEWAY_URL` and `AI_GATEWAY_PROVIDER` configured.
- Workflow with an agent_node that needs research skills.

**When**: Workflow executes a research agent_node.

**Then**: Verify:

1. Agent node receives system + user prompts correctly.
2. Request to `ai-gateway` includes correct model and `X-AI-Provider` header.
3. Gateway successfully translates and routes to the backend provider.
4. Agent receives and parses `tool_calls` from the gateway's OpenAI-compatible response.
5. Agent calls `load_skill("research")` via local MCP.
6. Agent submits tool result back to `ai-gateway` (ping-pong loop).
7. Agent returns findings with confidence score after loop completion.

## Test Flow

```
Step 1: Create workflow with ai-gateway configured agents
  - research_node: provider="github", model="gpt-4o", temperature=0.7
    - tools: ["load_skill", "search"]
  - verify_node: provider="anthropic", model="claude-3-5-sonnet", temperature=0.5
    - tools: ["verify"]

Step 2: Submit workflow via POST /execute
  - Assert 202 Accepted

Step 3: Monitor Gateway Proxy Activity
  - Verify research_node sends request to Gateway with:
    - URL: {AI_GATEWAY_URL}/chat/completions
    - Header: X-AI-Provider: github

Step 4: Verify the Tool Loop (Ping-Pong)
  - Assert assistant message with tool_calls received from Gateway.
  - Assert search() called by agent.
  - Assert subsequent request to Gateway includes role: "tool" and tool_call_id.
  - Assert Gateway returns final summarized response.

Step 5: Verify Provider Switching via Gateway
  - Verify verify_node sends request to Gateway with:
    - Header: X-AI-Provider: anthropic
  - Assert successful cross-provider translation by the Gateway.
```

## Acceptance Criteria

- [ ] `AIGatewayClient` correctly formats the multi-step tool result history.
- [ ] `X-AI-Provider` header matches the node's configuration in the proxy request.
- [ ] Agent successfully handles tool calls even when backend is Anthropic (translated by gateway).
- [ ] Happy path completes: submitted topic -> research findings -> verification -> finished state.
- [ ] All proxy interactions are logged in the runtime events.

## Environment Variables Required

| Variable              | Purpose                                         |
| --------------------- | ----------------------------------------------- |
| `AI_GATEWAY_URL`      | Proxy endpoint (e.g., http://localhost:8080/v1) |
| `AI_GATEWAY_PROVIDER` | Backend to test (e.g., 'github' or 'anthropic') |

## Test Coverage (8+ tests)

### Connectivity and Headers

- [ ] `test_e2e_gateway_connectivity` - health check before starting workflow.
- [ ] `test_e2e_header_propagation` - intercept proxy call to verify `X-AI-Provider`.

### Tool Loop Path

- [ ] `test_e2e_agent_tool_loop_completion` - verify the full ping-pong with the gateway.
- [ ] `test_e2e_translation_layer_validation` - use an Anthropic model via gateway to verify OpenAI-to-Anthropic translation works for tool calling.

### Error Scenarios

- [ ] `test_e2e_gateway_error_502` - simulate provider error and verify runtime retry.
- [ ] `test_e2e_gateway_auth_failure` - simulate missing token in gateway and verify 401 response handling.

## Implementation Notes

- Use the references in the [AI Gateway Integration Guide](file:///Users/destnguyxn/projects/monorepo/docs/ai-gateway/API_GUIDE.md) for expected status codes.
- Mock MCP responses for deterministic test outcomes.
- Use a dedicated test provider entry in `ai-gateway` if available.
