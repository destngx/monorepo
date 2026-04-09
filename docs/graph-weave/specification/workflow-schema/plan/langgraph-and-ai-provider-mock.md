# Plan: LangGraph Mock and AI Provider Mock Implementation [MOCK Phase]

## Objective

Implement real mock workflow execution for MOCK phase by:

1. Creating MockAIProvider that generates deterministic responses based on prompt type
2. Creating MockLangGraphExecutor that traverses workflow nodes and edges
3. Integrating executor into POST /execute and GET /execute/{run_id}/status endpoints
4. Tracking execution state through intermediate node states (not just "pending")

## Rationale

Current implementation returns "pending" immediately for all executions. Real execution simulation requires:

- Node-by-node traversal following the workflow JSON spec
- Edge routing based on deterministic conditions (JSONPath)
- AI provider calls that return realistic but consistent mock responses
- Execution event tracking for visibility into intermediate steps

This maintains the MOCK phase contract: externally integrated systems (AI, Redis, MCP) are mocked, but workflow orchestration is real.

## Architecture

```
POST /execute (request: ExecuteRequest)
  ├─> Load workflow from MockWorkflowStore
  ├─> Create run_id and initialize execution state
  ├─> Set current_run_id in MockLangGraphExecutor
  ├─> Call executor.execute(workflow, input_data, checkpoint_store, cache)
  │   ├─> Find entry node
  │   ├─> Loop: for each node (max_hops limit)
  │   │   ├─> If agent_node: call MockAIProvider with system + user prompts
  │   │   ├─> Store result in state.node_results[node_id]
  │   │   ├─> Find next node via edge routing (JSONPath conditions)
  │   │   └─> Log event to executor.execution_events[run_id]
  │   │   └─> Until exit node reached or limit exceeded
  │   └─> Return { run_id, status, events, final_state, hop_count }
  ├─> Store result in execution_runs[run_id]
  └─> Return ExecuteResponse(run_id, status, ...)

GET /execute/{run_id}/status
  └─> Return { run_id, status, events, final_state } from execution_runs[run_id]
```

## Key Decisions

### 1. Deterministic AI Provider Responses

- **Decision**: MockAIProvider routes to response templates based on keyword matching (search, sql, synthesis, classify, stagnation)
- **Why**: Deterministic = testable, repeatable, and consistent behavior for mock workflows
- **Alternative considered**: Random responses - rejected because unpredictable behavior makes testing workflow routing logic impossible

### 2. Execution State Tracking

- **Decision**: Each execution tracked in executor.execution_events[run_id] as list of timestamped events
- **Why**: Enables GET /status to return intermediate states instead of just "pending"; real visibility into workflow progress
- **State includes**: node_execute, agent_response, edge_route, node_exit, error events

### 3. Edge Condition Evaluation

- **Decision**: Simple JSONPath-like evaluation ($.field.subfield) with operators ==, !=, >, <
- **Why**: Aligns with WORKFLOW_JSON_SPEC.md contract; covers typical branching logic
- **Limitation**: Full JSONPath not needed for MOCK phase; MVP can extend if needed

### 4. Execution Run Storage

- **Decision**: execution_runs[run_id] = result dict (in-memory, lost on restart)
- **Why**: Consistent with MOCK phase mocks (in-memory, temporary)
- **MVP migration**: Replace with CheckpointStore persistence

## Dependencies

- MockAIProvider: New adapter (ai_provider.py)
- MockLangGraphExecutor: New executor (langgraph_executor.py)
- Existing workflow store, checkpoint store, cache adapters (no changes)
- Updated execute endpoint to call executor instead of returning immediate "pending"
- Updated status endpoint to return stored execution result

## Design Constraints

1. No actual LLM calls (MockAIProvider returns templates)
2. No actual Redis calls (cache is in-memory MockRedisAdapter)
3. Execution is synchronous (not async queue/worker; immediate return on POST /execute)
4. Node execution order is deterministic (first matching edge condition taken)
5. State mutations are logged as events but state itself is immutable during routing

## Implementation Phases (Red-Green-Refactor)

### RED: Test contracts for both adapters

- Test MockAIProvider returns consistent JSON for different prompt types
- Test MockLangGraphExecutor traverses nodes and edges correctly
- Test execution events are logged
- Test POST /execute stores result and GET /status retrieves it

### GREEN: Implement adapters and endpoints

- Create MockAIProvider with prompt routing
- Create MockLangGraphExecutor with node execution and edge routing
- Update execute endpoint to call executor
- Update status endpoint to return stored execution

### REFACTOR: Clean up, add verification tests

- Verify test coverage: node execution, edge routing, AI provider calls
- Verify execution events are correct
- Verify workflow state doesn't get corrupted
- Document patterns in delta-changes.md

## Testing Strategy

- Unit tests for MockAIProvider (prompt routing, response format)
- Unit tests for MockLangGraphExecutor (node execution, edge routing, condition evaluation)
- Integration tests for POST /execute → GET /status flow
- End-to-end test with real workflow JSON (financial-research example)

## Success Criteria

1. ✅ MockAIProvider exists and returns deterministic responses for 5+ prompt types
2. ✅ MockLangGraphExecutor traverses multi-node workflows correctly
3. ✅ Edge conditions (JSONPath) route to correct next nodes
4. ✅ Execution events logged for every step (node_execute, agent_response, edge_route, node_exit)
5. ✅ POST /execute returns run_id + status='running' (or 'completed' if sync)
6. ✅ GET /execute/{run_id}/status returns full execution events + final state
7. ✅ All tests pass (new + existing)
8. ✅ No deprecation warnings

## Deliverables

1. `apps/graph-weave/src/adapters/ai_provider.py` - MockAIProvider class
2. `apps/graph-weave/src/adapters/langgraph_executor.py` - MockLangGraphExecutor class
3. Updated `apps/graph-weave/src/main.py` - integrate executor into endpoints
4. Updated `tests/test_*.py` - verification tests for both adapters
5. Example workflow JSON with agent_node entries
6. Updated `docs/graph-weave/delta-changes.md` - decision log
7. Updated `docs/graph-weave/progress.md` - MOCK phase status

## Notes

- Both adapters are MOCK-phase final; MVP will replace with real LangGraph + real LLM provider
- No changes to existing MockWorkflowStore, MockCheckpointStore, MockRedisAdapter
- Execution is synchronous for simplicity; MVP can add async/queue model if needed
- Status endpoint always returns events list (enables tailing logs)
