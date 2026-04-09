# Verification: LangGraph Mock and AI Provider Mock [MOCK Phase]

Reference plan: [[../plan/langgraph-and-ai-provider-mock.md]]
Reference tasks: [[GW-MOCK-LANGGRAPH-TASKS.md]]

## Verification Strategy

This document defines the acceptance criteria and verification checklists for the LangGraph Mock and AI Provider Mock implementation.

## Test File Structure

```
tests/
├── test_ai_provider.py               (8+ tests)
├── test_langgraph_executor.py        (22+ tests)
├── test_workflow_execution_e2e.py    (4+ integration tests)
├── fixtures/
│   └── workflow_example_multi_node.json
└── conftest.py                       (fixtures for both adapters)
```

## MockAIProvider Verification (GW-MOCK-LANGGRAPH-001)

### Test Suite: test_ai_provider.py

**Test 1: Instantiation**

- Given: MockAIProvider()
- When: constructor called
- Then: instance created without error, \_call_count = 0

**Test 2: Call interface**

- Given: provider instance
- When: call(system_prompt, user_prompt, model="gpt-4")
- Then: returns dict with keys {content, tokens_used, model, call_count}

**Test 3: Research routing**

- Given: system_prompt contains "research"
- When: call() executed
- Then: content is JSON with {status, findings, sources, confidence}

**Test 4: SQL routing**

- Given: system_prompt contains "sql"
- When: call() executed
- Then: content is JSON with {status, query, rows, row_count, execution_time_ms}

**Test 5: Synthesis routing**

- Given: system_prompt contains "synthesize"
- When: call() executed
- Then: content is JSON with {status, summary, key_insights, risks, confidence}

**Test 6: Classification routing**

- Given: user_prompt contains "classify"
- When: call() executed
- Then: content is JSON with {status, classification, confidence, sub_types}

**Test 7: Stagnation routing**

- Given: user_prompt contains "stagnation"
- When: call() executed
- Then: content is JSON with {status, detected_stagnation, repeated_intents, recommendation}

**Test 8: Deterministic responses**

- Given: same (system_prompt, user_prompt) called twice
- When: call() executed both times
- Then: content is identical (deterministic)

**Test 9: Token estimation**

- Given: response with N characters
- When: call() executed
- Then: tokens_used ≈ N/4 (rough estimate)

**Test 10: Call count tracking**

- Given: provider.call() called 3 times
- When: inspect provider.\_call_count
- Then: \_call_count = 3

### Acceptance Criteria

- ✅ 8+ tests all passing
- ✅ 100% branch coverage for \_generate_response() routing
- ✅ All response types return valid JSON
- ✅ No deprecation warnings

---

## MockLangGraphExecutor Verification (GW-MOCK-LANGGRAPH-002 to GW-MOCK-LANGGRAPH-004)

### Test Suite: test_langgraph_executor.py

**Test 1: Instantiation**

- Given: MockLangGraphExecutor()
- When: constructor called with default MockAIProvider
- Then: instance created, ai_provider assigned, execution_events = {}

**Test 2: Node finding**

- Given: workflow with entry node, agent_node, exit node
- When: \_find_entry_node(workflow) called
- Then: returns entry node id

**Test 3: Entry node execution**

- Given: workflow starting at entry node
- When: execute() called with input_data
- Then: state initialized with {input, step, current_node, node_results}

**Test 4: Agent node execution**

- Given: agent_node with system_prompt and user_prompt_template
- When: \_execute_agent_node() called
- Then: MockAIProvider called, result stored in state.node_results[node_id]

**Test 5: Prompt interpolation**

- Given: user_prompt_template = "Analyze {topic} with depth {depth}"
- When: \_interpolate_prompt(template, {topic: "earnings", depth: 3})
- Then: returns "Analyze earnings with depth 3"

**Test 6: Exit node detection**

- Given: workflow with exit node
- When: execute() reaches exit node
- Then: execution stops, returns status='completed'

**Test 7: Max hops limit**

- Given: circular workflow (node loops to self)
- When: execute() with max_hops=5
- Then: stops after 5 hops, returns hop_count=5

**Test 8: Event logging (node_execute)**

- Given: execute() running
- When: agent_node executed
- Then: event logged with type='node_execute', timestamp, message

**Test 9: Event logging (agent_response)**

- Given: ai_provider returns response
- When: response stored
- Then: event logged with type='agent_response'

**Test 10: Event logging (edge_route)**

- Given: edge taken from node A to node B
- When: routing decision made
- Then: event logged with type='edge_route', from→to

**Test 11: Event logging (node_exit)**

- Given: exit node reached
- When: execute() completes
- Then: event logged with type='node_exit'

**Test 12: Condition evaluation (==)**

- Given: condition "$.status == 'research_complete'"
- When: \_evaluate_condition() called with matching state
- Then: returns True

**Test 13: Condition evaluation (!=)**

- Given: condition "$.confidence != 0.5"
- When: \_evaluate_condition() called with state.confidence = 0.9
- Then: returns True

**Test 14: Condition evaluation (>)**

- Given: condition "$.confidence > 0.8"
- When: \_evaluate_condition() called with state.confidence = 0.92
- Then: returns True

**Test 15: Condition evaluation (<)**

- Given: condition "$.hop_count < 20"
- When: \_evaluate_condition() called with state.hop_count = 5
- Then: returns True

**Test 16: Condition false match**

- Given: condition "$.status == 'research_complete'"
- When: \_evaluate_condition() called with state.status = 'error'
- Then: returns False

**Test 17: Edge routing (first match)**

- Given: edges with conditions [A==B, C==D]
- When: first condition true, second false
- Then: first edge taken

**Test 18: Edge routing (no match fallback)**

- Given: edges with conditions all false
- When: outgoing edges exist
- Then: first unconditional or first edge taken

**Test 19: No outgoing edges**

- Given: node with no outgoing edges
- When: routing from node
- Then: exit node returned

**Test 20: State value extraction**

- Given: path="$.node_results.agent_1.status"
- When: \_get_state_value(path, state) called
- Then: returns nested value correctly

**Test 21: Get events**

- Given: execution with 5 events logged
- When: get_events(run_id) called
- Then: returns list of 5 events with correct types

**Test 22: Run ID tracking**

- Given: set_current_run_id(run_id)
- When: execute() called
- Then: events stored under correct run_id

### Acceptance Criteria

- ✅ 22+ tests all passing
- ✅ Node execution tested
- ✅ Edge routing tested (all operators: ==, !=, >, <)
- ✅ Event logging tested (all event types)
- ✅ No deprecation warnings
- ✅ 100% branch coverage for routing logic

---

## Endpoint Integration Verification (GW-MOCK-LANGGRAPH-005 to GW-MOCK-LANGGRAPH-006)

### Test Suite: test_workflow_execution_e2e.py

**Test 1: POST /execute with valid workflow**

- Given: Valid ExecuteRequest with existing workflow_id
- When: POST /execute called
- Then: response has run_id, status='completed', workflow_id, tenant_id

**Test 2: POST /execute with missing workflow**

- Given: ExecuteRequest with non-existent workflow_id
- When: POST /execute called
- Then: response status=404 or error

**Test 3: GET /execute/{run_id}/status after POST**

- Given: run_id from POST /execute response
- When: GET /execute/{run_id}/status called
- Then: response has status='completed', events list, final_state

**Test 4: Status response includes all execution events**

- Given: workflow executed with 3 nodes
- When: GET /execute/{run_id}/status called
- Then: events includes node_execute, agent_response, edge_route, node_exit for each node

**Test 5: Final state includes node results**

- Given: executed workflow
- When: GET /execute/{run_id}/status called
- Then: final_state.node_results contains output from each agent_node

**Test 6: GET /execute/{run_id}/status with invalid run_id**

- Given: non-existent run_id
- When: GET /execute/{run_id}/status called
- Then: returns 404 or empty response

### Acceptance Criteria

- ✅ 4+ integration tests all passing
- ✅ POST /execute flows to GET /status without errors
- ✅ No test regressions in existing test suite

---

## Example Workflow Verification (GW-MOCK-LANGGRAPH-007)

**Fixture: workflow_example_multi_node.json**

Structure verification:

- ✅ Valid JSON syntax
- ✅ Has metadata: {name, version, description}
- ✅ Has 3+ nodes: entry, agent_node, exit
- ✅ Entry node type="entry" with config.properties
- ✅ Agent node type="agent_node" with system_prompt and user_prompt_template
- ✅ Exit node type="exit"
- ✅ 3+ edges connecting nodes in sequence
- ✅ Follows WORKFLOW_JSON_SPEC.md schema

Execution verification:

- ✅ Can load from MockWorkflowStore
- ✅ Can execute without errors
- ✅ Produces 5+ events
- ✅ Final state valid

---

## Full Test Suite Verification (GW-MOCK-LANGGRAPH-009)

**Command**: `bun nx run graph-weave:test`

**Acceptance Criteria**:

- ✅ Tests passed: 155+ (8 + 22 + 4 + existing 120+)
- ✅ Tests failed: 0
- ✅ Tests skipped: 1 (existing)
- ✅ Deprecation warnings: 0
- ✅ No regressions in existing tests
- ✅ Execution time < 30s

---

## Documentation Verification (GW-MOCK-LANGGRAPH-010)

**Files to verify**:

- ✅ plan/langgraph-and-ai-provider-mock.md exists and is complete
- ✅ tasks/GW-MOCK-LANGGRAPH-TASKS.md exists with 10 tasks
- ✅ verification/VERIFY-GW-MOCK-LANGGRAPH.md exists (this file)
- ✅ delta-changes.md has new entry with implementation decisions
- ✅ progress.md MOCK phase updated with status + metrics

---

## Regression Test Checklist

Before marking complete, verify no regressions:

- ✅ Existing workflow management tests pass (list, create, update, delete)
- ✅ Existing cache tests pass
- ✅ Existing error response tests pass (validation errors, 404s)
- ✅ All endpoints remain accessible
- ✅ OpenAPI docs still generated correctly
- ✅ Error handling for missing workflows works

---

## Success Definition

Implementation is **COMPLETE** when:

1. ✅ All 10 tasks completed
2. ✅ 155+ tests passing, 0 failed, 1 skipped
3. ✅ 0 deprecation warnings
4. ✅ No regressions
5. ✅ All verification checklists marked complete
6. ✅ Documentation updated
7. ✅ delta-changes.md entry recorded
