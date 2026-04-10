# GW-MVP-E2E-002: End-to-End LangGraph Agent Execution with MCP Tools

**Objective**: Verify real LangGraph agent execution calling real MCP tools (load_skill, search, verify), proving agents can load skills on demand and use external tools.

**Phase**: [MVP]

## Test Scenario

**Given**:

- Real LangGraph executor integrated
- Real MCP server providing: load_skill, search, verify tools
- Real Claude LLM (or configured test LLM)
- Workflow with agent_node that needs research skills

**When**: Workflow executes a research agent_node

**Then**: Verify:

1. Agent node receives system + user prompts correctly
2. Agent calls load_skill("research") via MCP
3. MCP returns skill definition (markdown with YAML)
4. Agent uses skill guidance to structure research
5. Agent calls search() MCP tool for web results
6. Agent returns findings with confidence score
7. Edge condition evaluates agent output (confidence > 0.7)
8. State includes research_output with correct schema

## Test Flow

```
Step 1: Create workflow with agent_node
  - system_prompt: "You are a researcher..."
  - user_prompt_template: "Research {topic}"
  - input_mapping: {topic: "AI ethics"}
  - output_key: "research_output"
  - output_schema with required fields: findings, confidence, sources

Step 2: Submit workflow via POST /execute
  - Assert 202 Accepted

Step 3: Poll until node.started for research node
  - Verify system_prompt and user_prompt are injected

Step 4: Wait for agent execution
  - Agent should call load_skill("research")
  - Assert MCP tool call logged in events
  - Assert search() called by agent
  - Agent produces findings with confidence

Step 5: Verify edge condition routing
  - If confidence > 0.7, route to verify node
  - If confidence <= 0.7, route back to research (retry loop)
  - Assert routing is deterministic

Step 6: Verify final output schema
  - Assert research_output includes findings, sources, confidence
  - Assert confidence is valid number (0-1)
  - Assert sources is array of URLs
```

## Acceptance Criteria

- [ ] Agent calls load_skill() and receives skill definition
- [ ] Agent calls search() and receives web results
- [ ] Agent output matches output_schema
- [ ] Edge conditions correctly evaluate and route
- [ ] State accumulates research_output in correct location
- [ ] All MCP tool calls are logged in event stream
- [ ] Test passes with real Claude LLM
- [ ] Execution latency reasonable (<10s for simple research)

## Deliverables

1. `tests/test_e2e_agent_execution_with_mcp.py` - New file (250+ lines)
   - `test_e2e_agent_calls_load_skill()` - skill loading verification
   - `test_e2e_agent_calls_search_tool()` - external tool calling
   - `test_e2e_agent_output_matches_schema()` - schema validation
   - `test_e2e_edge_condition_routes_based_on_output()` - routing verification
   - `test_e2e_agent_with_retry_loop()` - loop handling
   - `test_e2e_multi_agent_workflow()` - multiple agents in sequence

## Dependencies

- GW-MVP-RUNTIME-202: Real LangGraph Executor
- GW-MVP-RUNTIME-203: Event Emitter
- MCP servers (load_skill, search, verify) running

## Environment Variables Required

| Variable                   | Purpose                                                    | Loaded By                         |
| -------------------------- | ---------------------------------------------------------- | --------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for checkpoint storage and event logging    | Test fixture setup                |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token                               | Test fixture setup                |
| `GITHUB_TOKEN`             | GitHub API token for skill loading via Level 1 frontmatter | Test fixture setup, skill_service |

**Reference**: See `[[../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading in Tests**:

```python
# In tests/test_e2e_agent_execution_with_mcp.py
from src.config import Config
import pytest

@pytest.fixture(scope="session")
def redis_client():
    """Initialize Redis client with env vars from Config"""
    client = RedisClient(
        url=Config.UPSTASH_REDIS_REST_URL,
        token=Config.UPSTASH_REDIS_REST_TOKEN
    )
    yield client

@pytest.fixture(scope="session")
def skill_service():
    """Initialize skill service with GitHub token"""
    service = SkillService(github_token=Config.GITHUB_TOKEN)
    return service
```

## Test Coverage (6+ tests with comprehensive agent scenarios)

### Agent Execution Tests

- [ ] `test_e2e_agent_calls_load_skill` - agent loads research skill, receives definition
- [ ] `test_e2e_agent_calls_search_tool` - agent calls search(), gets web results
- [ ] `test_e2e_agent_output_matches_schema` - output validates against schema
- [ ] `test_e2e_edge_condition_routes_based_on_output` - confidence > 0.7 routes correctly

### Multi-Agent Tests

- [ ] `test_e2e_multi_agent_workflow` - sequential agents in same workflow
- [ ] `test_e2e_agent_with_retry_loop` - low confidence triggers retry loop

### Tool Routing Tests

- [ ] `test_e2e_all_mcp_tools_called_correctly` - load_skill, search, verify all work
- [ ] `test_e2e_tool_responses_formatted_correctly` - responses embedded in agent feedback

### Error Scenarios

- [ ] `test_e2e_agent_malformed_tool_call` - invalid tool name handled gracefully
- [ ] `test_e2e_agent_missing_tool_arguments` - missing args trigger retry
- [ ] `test_e2e_agent_api_timeout` - LLM timeout handled with retry

### Concurrency Tests

- [ ] `test_e2e_concurrent_agent_workflows` - 3+ agents executing in parallel
- [ ] `test_e2e_agent_tool_calls_idempotent` - same tool call → cached response

### Performance Tests

- [ ] Agent execution <10s for simple research task
- [ ] Tool response caching reduces latency on repeated queries
- [ ] MCP routing overhead <100ms per tool call

## Implementation Notes

- Mock MCP responses deterministically for consistency
- Use TestLLM class instead of Claude for faster testing
- Verify MCP tool calls in event stream (tool.called, tool.result events) via Redis
- Assert output_schema validation passes
- Use fixtures for test workflows
- **All tests MUST have UPSTASH and GITHUB env vars loaded before execution**
- Checkpoint storage used for agent recovery paths
- Concurrency: Verify no interference between parallel agent executions
- Tool caching: Same query within 5 minutes returns cached result
