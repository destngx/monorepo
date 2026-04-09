# Tasks: LangGraph Mock and AI Provider Mock [MOCK Phase]

Reference plan: [[langgraph-and-ai-provider-mock.md]]

## GW-MOCK-LANGGRAPH-001: Implement MockAIProvider adapter

**Acceptance Criteria**:

- MockAIProvider class instantiates without errors
- call(system_prompt, user_prompt) returns dict with {content, tokens_used, model, call_count}
- Content is valid JSON string
- Routing works for 5+ prompt types: research, sql, synthesis, classification, stagnation
- Same prompt always returns same response (deterministic)
- Response fields match: status, findings/query/summary/confidence based on type

**Test coverage**: 8+ tests in test_ai_provider.py

**Deliverable**: `apps/graph-weave/src/adapters/ai_provider.py`

---

## GW-MOCK-LANGGRAPH-002: Implement MockLangGraphExecutor node execution

**Acceptance Criteria**:

- MockLangGraphExecutor class instantiates with optional MockAIProvider
- execute() method signature: execute(workflow, input_data, checkpoint_store, cache) → dict
- Finds entry node and starts execution there
- Executes agent_node: calls ai_provider with system_prompt + user_prompt_template
- Stores result in state.node_results[node_id]
- Returns final execution result with {run_id, status, events, final_state, hop_count}
- Respects max_hops limit (default 20)
- Stops at exit node

**Test coverage**: 12+ tests in test_langgraph_executor.py

**Deliverable**: `apps/graph-weave/src/adapters/langgraph_executor.py`

---

## GW-MOCK-LANGGRAPH-003: Implement MockLangGraphExecutor edge routing

**Acceptance Criteria**:

- \_route_by_edge() method finds outgoing edges from current node
- Evaluates conditions: ==, !=, >, < with JSONPath-like syntax ($.field.subfield)
- First matching condition edge is taken
- If no conditions match, takes first unconditional edge
- Logs edge_route event with from→to node info
- Returns next node_id

**Test coverage**: 6+ tests for condition evaluation (==, !=, >, <, no match)

**Deliverable**: Part of langgraph_executor.py

---

## GW-MOCK-LANGGRAPH-004: Implement MockLangGraphExecutor event logging

**Acceptance Criteria**:

- \_log_event(run_id, event_type, message) creates timestamped event
- Event types: node_execute, agent_response, edge_route, node_exit, node_entry, error, warning
- Events stored in execution_events[run_id] as list
- get_events(run_id) returns list of all events for a run
- Each event has {timestamp, type, message}
- Timestamps in ISO8601 format with Z suffix

**Test coverage**: 4+ tests for event logging

**Deliverable**: Part of langgraph_executor.py

---

## GW-MOCK-LANGGRAPH-005: Update POST /execute endpoint to call executor

**Acceptance Criteria**:

- POST /execute loads workflow from MockWorkflowStore
- Returns 404 if workflow not found
- Calls langgraph_executor.execute(workflow, input_data, checkpoint_store, cache)
- Stores result in execution_runs[run_id]
- Returns ExecuteResponse(run_id, status='completed' or 'error', ...)
- Logs execution start and completion with run_id

**Test coverage**: 4+ tests (workflow found, workflow not found, execution success, execution error)

**Deliverable**: Updated `apps/graph-weave/src/main.py` execute endpoint

---

## GW-MOCK-LANGGRAPH-006: Update GET /execute/{run_id}/status endpoint

**Acceptance Criteria**:

- GET /execute/{run_id}/status returns stored execution result from execution_runs[run_id]
- Returns 404 if run_id not found
- Response includes: {run_id, status, events, final_state, hop_count}
- Final state includes: {input, step, current_node, node_results, status, hop_count}
- Events list is complete with all intermediate steps

**Test coverage**: 3+ tests (status found, status not found, events are correct)

**Deliverable**: Updated `apps/graph-weave/src/main.py` status endpoint

---

## GW-MOCK-LANGGRAPH-007: Create test example workflow

**Acceptance Criteria**:

- Create workflow JSON with 3+ nodes: entry → agent_node → exit
- Agent node has system_prompt + user_prompt_template
- Entry node has input configuration
- Nodes connected by edges with no conditions
- Follows WORKFLOW_JSON_SPEC.md structure
- Can be loaded by MockWorkflowStore
- Suitable for integration testing

**Test coverage**: 1 integration test using this workflow

**Deliverable**: `apps/graph-weave/tests/fixtures/workflow_example_multi_node.json`

---

## GW-MOCK-LANGGRAPH-008: Integration test: POST execute + GET status flow

**Acceptance Criteria**:

- Create workflow and store in MockWorkflowStore
- POST /execute with valid request
- Assert response has run_id, status, workflow_id, tenant_id
- GET /execute/{run_id}/status
- Assert status is 'completed'
- Assert events list has 5+ entries (entry, execute, response, route, exit)
- Assert final_state.node_results has results from agent nodes

**Test coverage**: 1 end-to-end test

**Deliverable**: Part of test_workflow_execution.py

---

## GW-MOCK-LANGGRAPH-009: Add verification tests for deprecation warnings

**Acceptance Criteria**:

- Run test suite: `bun nx run graph-weave:test`
- Assert no deprecation warnings from Pydantic v2 or FastAPI
- Assert all 140+ tests pass (including new tests)
- Record metrics: test count, coverage, warnings

**Test coverage**: 1 verification run

**Deliverable**: CI/verification pass documented in delta-changes.md

---

## GW-MOCK-LANGGRAPH-010: Update documentation

**Acceptance Criteria**:

- Add plan/langgraph-and-ai-provider-mock.md
- Add tasks/GW-MOCK-LANGGRAPH-\*.md
- Update delta-changes.md with implementation decisions and any friction
- Update progress.md MOCK phase row: add completion status, test count, notes
- All docs reference the new adapters and their roles

**Test coverage**: Manual review of doc completeness

**Deliverable**: Updated spec docs + delta-changes.md + progress.md

---

## Acceptance Definition

All tasks completed when:

1. ✅ MockAIProvider tests pass (8+)
2. ✅ MockLangGraphExecutor tests pass (22+)
3. ✅ Endpoint integration tests pass (4+)
4. ✅ End-to-end workflow test passes (1+)
5. ✅ Total test suite passes (155+ tests, 0 failed, 0 warnings)
6. ✅ Documentation updated with full context
7. ✅ No regressions in existing tests
