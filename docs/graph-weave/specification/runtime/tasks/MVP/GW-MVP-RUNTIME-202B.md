# GW-MVP-RUNTIME-202B: LLM & MCP Tool Integration

**Objective**: Integrate GitHub Copilot AI provider with MCP tool routing (load_skill, search, verify) for agent decision making.

**Phase**: [MVP]

**Duration**: 1.5 hours

**Blocked By**: DATA-201A, DATA-201B

**Blocks**: RUNTIME-202C

## Requirements

### Functional

- GitHub Copilot API integration (token-based authentication via GITHUB_TOKEN)
- **Per-node provider routing**: support multiple LLM providers (GitHub Copilot, OpenAI, etc.)
- **Per-node model override**: each agent_node can specify custom model
- Per-node temperature and max_tokens override
- System + user prompt formatting (from node config)
- Tool call parsing from LLM responses
- MCP tool routing: load_skill, search, verify (subset per node's tools list)
- Tool response formatting for agent feedback
- Streaming support (for token-level feedback)
- Graceful handling of malformed tool calls
- Retry on transient API failures

### Non-Functional

- LLM API calls <5s for typical queries (per provider)
- Tool routing deterministic (same input → same tool call)
- Memory efficient (no unnecessary context retention)
- Tool response caching (same query → cached response)
- Support for multiple providers simultaneously (different nodes can use different providers)

## Implementation Approach

1. Create `src/adapters/mcp_router.py`:
   - `MCPRouter` class:
     - `load_skill(skill_name)` → MarkdownContent
     - `search(query)` → SearchResults
     - `verify(claim)` → VerificationResult
   - Tool call handlers with error recovery
   - Response formatting for LLM consumption
   - Response caching decorator
   - **Provider router**:
     - `get_provider_client(provider_name, model_name)` → LLM client
     - Support GitHub Copilot (github), OpenAI (openai), etc.
     - Validate provider API keys loaded (from .env)

2. Update `src/adapters/langgraph_executor.py`:
   - Agent node function now receives per-node config:
     - `provider`, `model`, `temperature`, `max_tokens`, `tools`
   - Route to appropriate LLM provider based on node config
   - Format system + user prompts using node-specific templates
   - Parse tool calls from LLM response
   - Filter tool calls by node's allowed tools list
   - Route through MCPRouter
   - Retry on failure
   - Accumulate results in state

3. Provider configuration:

   ```python
   PROVIDER_CONFIGS = {
     "github": {
       "required_env": "GITHUB_TOKEN",
       "models": ["claude-3.5-sonnet", "claude-3-opus", ...],
       "default_model": "claude-3.5-sonnet"
     },
     "openai": {
       "required_env": "OPENAI_API_KEY",
       "models": ["gpt-4", "gpt-4-turbo", ...],
       "default_model": "gpt-4"
     }
   }
   ```

4. Implementation notes:
   - Use GitHub Copilot SDK (via GITHUB_TOKEN authentication)
   - Mock provider responses in tests (don't make real API calls)
   - Tool results cached in Redis via RedisClient
   - Streaming setup for token-level feedback (for future)
   - Node-level tools list constrains which MCP tools can be called

## Acceptance Criteria

- [ ] GitHub Copilot API integrated (not mock)
- [ ] Tool calls parsed correctly from GitHub Copilot responses
- [ ] MCP tools called for load_skill, search, verify
- [ ] Tool responses formatted for agent feedback
- [ ] Streaming support implemented
- [ ] Error handling: malformed tool calls logged
- [ ] All tests passing (15+ tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-050 [MVP]: Agent must use real MCP tools for skill loading, search, verification
- FR-RUNTIME-041 [MVP,FULL]: Level 1 skill frontmatter must always be loaded before routing

## Deliverables

1. `src/adapters/mcp_router.py` (120 LOC)
2. Update `src/adapters/langgraph_executor.py` (100 LOC)
3. `tests/test_mcp_router.py` (200+ LOC, 15+ tests)

## Test Coverage (15+ tests)

### Unit Tests

- [ ] MCPRouter.load_skill("research") returns markdown
- [ ] MCPRouter.search("quantum computing") returns results
- [ ] MCPRouter.verify("Earth is flat") returns verdict
- [ ] Tool response formatting: markdown preserved
- [ ] Tool response formatting: JSON cleaned
- [ ] Prompt formatting: system prompt included
- [ ] Prompt formatting: variables interpolated
- [ ] Tool call parsing: extract tool_name and arguments
- [ ] Tool call parsing: malformed response handled
- [ ] **Provider routing: GitHub Copilot provider returns Copilot client**
- [ ] **Provider routing: OpenAI provider returns OpenAI client**
- [ ] **Provider API key validation: missing GITHUB_TOKEN raises error**
- [ ] **Model validation: unknown model for provider raises error**
- [ ] **Temperature validation: value outside [0,1] raises error**
- [ ] **Tools list filtering: only allowed tools called**

### Integration Tests (with LLM API keys)

- [ ] LLM call succeeds with valid prompt via GitHub Copilot
- [ ] LLM call succeeds with valid prompt via OpenAI
- [ ] LLM response includes expected tool calls
- [ ] Tool routing: load_skill called → SkillService invoked
- [ ] Tool routing: search called → search service invoked
- [ ] Tool routing: verify called → verification service invoked
- [ ] Error handling: API rate limit caught and retried
- [ ] Error handling: Invalid API key caught
- [ ] **Provider switching: same workflow can use multiple providers in different nodes**
- [ ] **Per-node temperature affects response consistency**

## Error Scenarios

- [ ] Invalid tool name in LLM response → logged, retry
- [ ] Tool call missing required arguments → logged, agent retry
- [ ] LLM API timeout (>10s) → retry once, then error
- [ ] MCP service unavailable → tool fails gracefully
- [ ] Malformed JSON in response → parsed safely
- [ ] **Invalid provider name → error with supported providers list**
- [ ] **Invalid model for provider → error with supported models list**

## Environment Variables

- `UPSTASH_REDIS_REST_URL`
- `UPSTASH_REDIS_REST_TOKEN`
- `GITHUB_TOKEN` (GitHub Copilot authentication)
- `OPENAI_API_KEY` (if using OpenAI provider)

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy.

## Implementation Notes

- Use GitHub Copilot SDK with GITHUB_TOKEN (token-based authentication)
- Mock MCP services in tests (use fixtures)
- Implement tool result caching (same query → cached response)
- Response formatting: preserve markdown structure
- This task depends on DATA-201A/B (Redis for caching)
- This task blocks RUNTIME-202C (executor needs this)
- Prerequisite for E2E-002 (agent execution testing)

## Testing Strategy

- Unit: Tool response formatting, prompt formatting, parsing
- Integration: Real GitHub Copilot API (use test token), mock MCP services
- Error scenarios: API failures, malformed responses, service unavailability
- Caching: Verify same query returns cached result
- Streaming: Setup only (full streaming in later phases)
