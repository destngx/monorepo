# GW-MVP-RUNTIME-202C: Executor & Stagnation Detection

**Objective**: Execute real LangGraph with event emission and stagnation detection for interruption safety.

**Phase**: [MVP]

**Duration**: 1 hour

**Blocked By**: RUNTIME-202A, RUNTIME-202B

**Blocks**: DATA-202, E2E-002, E2E-003

## Requirements

### Functional

- Real execution loop through StateGraph
- Event emission after each node: node.started, node.completed, node.failed
- Stagnation detection: max 20 node visits (configurable)
- Execution timeout: 300 seconds (configurable)
- Circuit breaker: checks kill flag after each node
- State accumulation: all node outputs merged into workflow_state
- Event persistence: all events stored in Redis immediately
- Execution resumable: can continue from checkpoint

### Non-Functional

- Execution latency <5s for simple 3-node workflow
- Memory usage <200MB per concurrent execution
- No blocking calls in main loop
- Events append-only (immutable log)

## Implementation Approach

1. Create `src/adapters/stagnation_detector.py`:
   - `StagnationDetector` class:
     - `track_node_visit(node_id)` → visit_count
     - `is_stagnated(max_hops=20)` → bool
     - `reset()` for new workflow
     - Visit counter: `{node_id: count}`

2. Finalize `src/adapters/langgraph_executor.py`:
   - `RealLangGraphExecutor.execute(run_id, thread_id, tenant_id, workflow, input_data)`
   - Main execution loop:
     1. Get current node
     2. Check kill flag (circuit breaker)
     3. Check stagnation
     4. Execute node
     5. Emit events (node.started, node.completed)
     6. Update workflow_state
     7. Move to next node via edge condition
     8. Repeat until exit node or timeout
   - Timeout enforcement: start timer, check after each node
   - Event emission: append to Redis event log
   - Error handling: node failures emit node.failed, continue to exit

3. Implementation notes:
   - Use `time.monotonic()` for timeout tracking (not wall clock)
   - Node visit tracking: `dict {node_id: count}`
   - All events persisted to Redis immediately (async OK)
   - Kill flag stored in Redis, checked after each node
   - State accumulation: merge outputs into workflow_state

## Acceptance Criteria

- [ ] Execution completes successfully for happy path
- [ ] Events emitted in correct order (node.started → node.completed)
- [ ] Stagnation detection stops execution after 20 visits
- [ ] Timeout stops execution after 300s
- [ ] Circuit breaker: kill flag stops execution
- [ ] State accumulation: all outputs in final state
- [ ] All tests passing (15+ tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-060 [MVP]: Execution loop must emit granular events
- FR-RUNTIME-061 [MVP]: Stagnation detection prevents infinite loops
- FR-RUNTIME-062 [MVP]: Timeout enforcement prevents runaway execution

## Deliverables

1. `src/adapters/stagnation_detector.py` (80 LOC)
2. Finalize `src/adapters/langgraph_executor.py` (300 LOC total)
3. `tests/test_stagnation_detector.py` (150+ LOC, 15+ tests)

## Test Coverage (15+ tests)

### Unit Tests

- [ ] StagnationDetector tracks visit count per node
- [ ] StagnationDetector.is_stagnated() returns True at max_hops
- [ ] StagnationDetector.is_stagnated() returns False below threshold
- [ ] Timeout countdown works correctly
- [ ] Event emission format correct: `{type, run_id, timestamp, data}`

### Integration Tests (with StateGraph)

- [ ] 3-node workflow executes start-to-finish
- [ ] Events emitted: request.started, node.started x3, node.completed x3, request.completed
- [ ] Workflow state accumulates all node outputs
- [ ] State available at GET /execute/{run_id}/status
- [ ] Stagnation: infinite loop detected and execution halted
- [ ] Timeout: 300s execution stopped
- [ ] Circuit breaker: kill flag stops execution mid-workflow
- [ ] Concurrent execution: multiple workflows track visits independently

## Concurrency Tests

- [ ] Multiple workflows execute concurrently without interference
- [ ] Visit counters remain independent across workflows
- [ ] Timeout tracking works correctly per execution

## Error Scenarios

- [ ] Node execution fails → node.failed emitted, execution continues to exit
- [ ] Node throws exception → caught, logged, workflow fails gracefully
- [ ] Kill flag set during execution → immediate halt
- [ ] Stagnation detected → execution.stagnated event, workflow halts
- [ ] Timeout triggered → execution.timeout event, workflow halts
- [ ] Node returns no output → state passed through unchanged

## Environment Variables

- `UPSTASH_REDIS_REST_URL`
- `UPSTASH_REDIS_REST_TOKEN`
- `ANTHROPIC_API_KEY` or `OPENAI_API_KEY`

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy.

## Implementation Notes

- Use `time.monotonic()` for timeout tracking (not wall clock)
- Node visit tracking: `dict {node_id: count}`
- All events persisted to Redis immediately (can be async)
- Kill flag: check Redis after each node
- Stagnation: global per workflow (track all visits)
- This task depends on:
  - RUNTIME-202A (graph builder)
  - RUNTIME-202B (LLM integration)
- Blocks E2E-002 and E2E-003 (full execution testing)

## Testing Strategy

- Unit: Stagnation detection, timeout tracking, event formatting
- Integration: Full workflow execution with real graph
- Concurrency: Multiple workflows under load
- Error scenarios: Node failures, stagnation, timeout, kill flag
- Performance: Verify execution latency <5s for simple workflows
