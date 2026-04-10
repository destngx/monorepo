# GW-MVP-RUNTIME-202B: LLM & MCP Tool Integration

**Objective**: Integrate Claude LLM with MCP tool routing (load_skill, search, verify) for agent decision making.

**Phase**: [MVP]

**Duration**: 1.5 hours

**Blocked By**: DATA-201A, DATA-201B

**Blocks**: RUNTIME-202C

## Requirements

### Functional

- Claude API integration (model configurable via .env)
- System + user prompt formatting
- Tool call parsing from Claude responses
- MCP tool routing: load_skill, search, verify
- Tool response formatting for agent feedback
- Streaming support (for token-level feedback)
- Graceful handling of malformed tool calls
- Retry on transient API failures

### Non-Functional

- Claude API calls <5s for typical queries
- Tool routing deterministic (same input → same tool call)
- Memory efficient (no unnecessary context retention)
- Tool response caching (same query → cached response)

## Implementation Approach

1. Create `src/adapters/mcp_router.py`:
   - `MCPRouter` class:
     - `load_skill(skill_name)` → MarkdownContent
     - `search(query)` → SearchResults
     - `verify(claim)` → VerificationResult
   - Tool call handlers with error recovery
   - Response formatting for LLM consumption
   - Response caching decorator

2. Update `src/adapters/langgraph_executor.py`:
   - Agent node function:
     - Format system + user prompts
     - Call Claude API
     - Parse tool calls
     - Route through MCPRouter
     - Retry on failure
     - Accumulate results in state

3. Implementation notes:
   - Use Anthropic SDK or LangChain for Claude
   - Mock MCP services in tests (don't make real API calls)
   - Tool results cached in Redis via RedisClient
   - Streaming setup for token-level feedback (for future)

## Acceptance Criteria

- [ ] Claude API integrated (not mock)
- [ ] Tool calls parsed correctly from Claude responses
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
- [ ] Claude prompt formatting: system prompt included
- [ ] Claude prompt formatting: variables interpolated
- [ ] Tool call parsing: extract tool_name and arguments
- [ ] Tool call parsing: malformed response handled

### Integration Tests (with Claude API key)

- [ ] Claude API call succeeds with valid prompt
- [ ] Claude response includes expected tool calls
- [ ] Tool routing: load_skill called → SkillService invoked
- [ ] Tool routing: search called → search service invoked
- [ ] Tool routing: verify called → verification service invoked
- [ ] Error handling: API rate limit caught and retried
- [ ] Error handling: Invalid API key caught

## Error Scenarios

- [ ] Invalid tool name in Claude response → logged, retry
- [ ] Tool call missing required arguments → logged, agent retry
- [ ] Claude API timeout (>10s) → retry once, then error
- [ ] MCP service unavailable → tool fails gracefully
- [ ] Malformed JSON in response → parsed safely

## Environment Variables

- `UPSTASH_REDIS_REST_URL`
- `UPSTASH_REDIS_REST_TOKEN`
- `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` (Claude)
- `GITHUB_TOKEN` (skill loading)

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy.

## Implementation Notes

- Use Anthropic SDK for Claude (not LangChain initially)
- Mock MCP services in tests (use fixtures)
- Implement tool result caching (same query → cached response)
- Response formatting: preserve markdown structure
- This task depends on DATA-201A/B (Redis for caching)
- This task blocks RUNTIME-202C (executor needs this)
- Prerequisite for E2E-002 (agent execution testing)

## Testing Strategy

- Unit: Tool response formatting, prompt formatting, parsing
- Integration: Real Claude API (use test key), mock MCP services
- Error scenarios: API failures, malformed responses, service unavailability
- Caching: Verify same query returns cached result
- Streaming: Setup only (full streaming in later phases)
