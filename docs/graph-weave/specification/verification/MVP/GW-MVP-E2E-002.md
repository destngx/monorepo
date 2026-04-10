# GW-MVP-E2E-002: End-to-End LangGraph Agent Execution with MCP Tools

**Objective**: Verify real LangGraph agent execution calling real MCP tools (load_skill, search, verify), proving agents can load skills on demand and use external tools.

**Phase**: [MVP]

## Test Scenario

**Given**:

- Real LangGraph executor integrated
- Real MCP server providing: load_skill, search, verify tools
- Real GitHub Copilot LLM (or configured test LLM)
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
Step 1: Create workflow with multi-provider agent nodes
  - research_node: provider="github", model="claude-3.5-sonnet", temperature=0.7
    - system_prompt: "You are a researcher..."
    - user_prompt_template: "Research {topic}"
    - tools: ["load_skill", "search"]
    - output_key: "research_output"
    - output_schema: {findings, confidence, sources}
  - verify_node: provider="github", model="claude-3.5-sonnet", temperature=0.5
    - system_prompt: "You are a fact checker..."
    - tools: ["verify"]
    - output_key: "verification_result"

Step 2: Submit workflow via POST /execute
  - Assert 202 Accepted

Step 3: Poll until node.started for research node
  - Verify system_prompt and user_prompt are injected
  - Verify GitHub Copilot provider will be used (from node config)

Step 4: Wait for research agent execution
  - Agent should call GitHub Copilot API (not other providers)
  - Assert load_skill("research") called
  - Assert search() called by agent
  - Agent produces findings with confidence
  - Assert MCP tool call logged in events

Step 5: Verify provider switching (edge condition routes to verify_node)
  - If confidence > 0.7, route to verify node (GitHub Copilot provider)
  - Verify GitHub Copilot API called (consistent provider)
  - Assert verify() called (allowed by verify_node tools list)
  - Assert search() NOT available to verify_node

Step 6: Verify final output schema
  - Assert research_output includes findings, sources, confidence
  - Assert verification_result from verify_node
  - Both outputs in correct state locations
  - Confidence is valid number (0-1)
  - Sources is array of URLs
```

## Acceptance Criteria

- [ ] Agent calls load_skill() and receives skill definition
- [ ] Agent calls search() and receives web results
- [ ] Agent output matches output_schema
- [ ] Edge conditions correctly evaluate and route
- [ ] State accumulates research_output in correct location
- [ ] All MCP tool calls are logged in event stream
- [ ] Test passes with real GitHub Copilot LLM
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

| Variable                   | Purpose                                                 | Loaded By                         |
| -------------------------- | ------------------------------------------------------- | --------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for checkpoint storage and event logging | Test fixture setup                |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token                            | Test fixture setup                |
| `GITHUB_TOKEN`             | GitHub API token for skill loading and Copilot access   | Test fixture setup, skill_service |

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

## Test Coverage (8+ tests with provider switching)

### Per-Provider Agent Tests

- [ ] `test_e2e_agent_with_github_copilot_provider` - verify GitHub Copilot API called
- [ ] `test_e2e_per_node_temperature_effects` - different temps produce different responses
- [ ] `test_e2e_per_node_tools_constrained` - node cannot call tools not in its list
- [ ] `test_e2e_per_node_max_tokens_respected` - responses shorter with lower max_tokens
- [ ] `test_e2e_multi_node_github_copilot_workflow` - multiple nodes using GitHub Copilot

### Per-Node Config Tests

- [ ] `test_e2e_agent_calls_load_skill` - skill loading verification
- [ ] `test_e2e_agent_calls_search_tool` - external tool calling
- [ ] `test_e2e_agent_output_matches_schema` - schema validation
- [ ] `test_e2e_edge_condition_routes_based_on_output` - routing verification
- [ ] `test_e2e_agent_with_retry_loop` - loop handling
- [ ] `test_e2e_multi_agent_workflow` - multiple agents with different configs

### Error Scenarios

- [ ] `test_e2e_invalid_provider_in_node_config` - missing provider error
- [ ] `test_e2e_invalid_model_for_github_provider` - model validation error
- [ ] `test_e2e_missing_github_token` - token validation error
- [ ] `test_e2e_tool_not_in_node_tools_list` - tool access control

### Tool Routing Tests

- [ ] `test_e2e_all_mcp_tools_called_correctly` - load_skill, search, verify all work
- [ ] `test_e2e_tool_responses_formatted_correctly` - responses embedded in agent feedback

### Concurrency Tests

- [ ] `test_e2e_concurrent_agent_workflows` - 3+ agents using GitHub Copilot
- [ ] `test_e2e_agent_tool_calls_idempotent` - same tool call → cached response

### Performance Tests

- [ ] Agent execution <10s for simple research task
- [ ] Tool response caching reduces latency on repeated queries
- [ ] MCP routing overhead <100ms per tool call
- [ ] GitHub Copilot API overhead <500ms per call

## Implementation Notes

- Mock MCP responses deterministically for consistency
- Use TestLLM class instead of GitHub Copilot for faster testing
- Verify MCP tool calls in event stream (tool.called, tool.result events) via Redis
- Assert output_schema validation passes
- Use fixtures for test workflows
- **All tests MUST have UPSTASH and GITHUB env vars loaded before execution**
- Checkpoint storage used for agent recovery paths
- Concurrency: Verify no interference between parallel agent executions
- Tool caching: Same query within 5 minutes returns cached result
