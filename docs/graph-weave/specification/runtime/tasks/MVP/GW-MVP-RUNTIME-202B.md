# GW-MVP-RUNTIME-202B: AI Gateway Proxy & MCP Tool Integration

**Objective**: Integrate the `ai-gateway` proxy for LLM interactions and MCP tool routing. This decouples the runtime from specific provider SDKs and leverages the gateway's unified OpenAI-compatible API.

**Phase**: [MVP]

**Duration**: 1.5 hours

**Blocked By**: DATA-201A, DATA-201B

**Blocks**: RUNTIME-202C

## Requirements

### Functional

- **Unified API Client**: Use a standard OpenAI-compatible client pointing to the `ai-gateway` base URL.
- **Provider Header Propagation**: Inject the `X-AI-Provider` header into every request based on node configuration.
- **Per-node model override**: Each `agent_node` can specify a custom model name recognized by the proxy backend.
- **System + user prompt formatting**: Consistent with OpenAI message format.
- **Tool call parsing**: Extract `tool_calls` from the gateway's OpenAI-compatible response.
- **MCP tool routing**: Route recognized tools (`load_skill`, `search`, `verify`) to the local `MCPRouter`.
- **Tool Result Loop**: Implement the multi-step "ping-pong" interaction (role: "tool", `tool_call_id`) as specified in the `AI_GATEWAY_GUIDE`.
- **Streaming support**: Handle server-sent events (SSE) from the gateway.

### Non-Functional

- **Authentication Statelessness**: No provider-specific API keys stored in GraphWeave; authentication is handled by the gateway.
- **Low Latency**: Gateway overhead should be <50ms p99.
- **Retry Logic**: Handle 502/504 errors from the gateway with exponential backoff.

## Implementation Approach

1. **Create `src/adapters/ai_gateway_adapter.py`**:
   - `AIGatewayClient` class:
     - Initialize with `base_url` (e.g., `http://localhost:8080/v1`).
     - `chat_completion(messages, tools, provider, model, ...)` method.
     - Automatically injects `X-AI-Provider` and `Content-Type: application/json`.
     - Handles streaming vs. sync response parsing.

2. **Update `src/adapters/mcp_router.py`**:
   - Maintain the tool execution logic for `load_skill`, `search`, and `verify`.
   - Ensure output is formatted into the `content` string expected by the OpenAI `tool` role.

3. **Update `src/adapters/langgraph_executor.py`**:
   - Agent nodes now use the `AIGatewayClient`.
   - Implement the tool loop:
     1. Send prompt + tools to Gateway.
     2. If `tool_calls` returned:
        a. Loop through tool calls, execute via `MCPRouter`.
        b. Append assistant message (with `tool_calls`) and tool results (role: "tool") to history.
        c. Re-submit to Gateway.
     3. Return final content output.

## Acceptance Criteria

- [ ] `AIGatewayClient` successfully routes calls through the proxy on `port 8080`.
- [ ] `X-AI-Provider` header is correctly set from node configuration.
- [ ] Tool calls are parsed correctly from the Gateway's translated response (e.g., even if backend is Anthropic).
- [ ] Tool loop (ping-pong) completes successfully for complex multi-tool queries.
- [ ] Error handling: 502/504 status codes from the gateway triggered retries.
- [ ] Verification evidence: Successful routing to 'github' and 'anthropic' backends via gateway.

## Related Requirements

- FR-RUNTIME-050 [MVP]: Use the `ai-gateway` proxy for all LLM-driven decision nodes.
- FR-GATEWAY-001 [FULL]: Reference the `[[../../ai-gateway/API_GUIDE]]` for protocol compliance.

## Environment Variables

| Variable              | Purpose                                                   |
| --------------------- | --------------------------------------------------------- |
| `AI_GATEWAY_URL`      | Base URL of the proxy (default: http://localhost:8080/v1) |
| `DEFAULT_AI_PROVIDER` | Fallback backend if not specified in node config.         |

## Test Coverage (15+ tests)

### Unit Tests

- [ ] `AIGatewayClient` generates correct headers for different providers.
- [ ] `AIGatewayClient` formats tool results according to OpenAI spec (`tool_call_id`).
- [ ] `MCPRouter` produces strings compatible with the gateway's "content" field.

### Integration Tests (requires running ai-gateway)

- [ ] Connectivity check: Gateway `/health` is reachable.
- [ ] Happy path: Single prompt to `github` provider via proxy returns text.
- [ ] Tool path: "What is the weather?" triggers `get_weather` call through proxy.
- [ ] Provider switching: Verify different nodes hit different backends via same gateway client.

## Reference Documents

- [AI Gateway API Guide](file:///Users/destnguyxn/projects/monorepo/docs/ai-gateway/API_GUIDE.md)
- [Tool Calling Guide](file:///Users/destnguyxn/projects/monorepo/docs/ai-gateway/tools/TOOL_CALLING_GUIDE.md)
