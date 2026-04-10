# GW-MVP-E2E-001: End-to-End Workflow Submission and Execution

**Objective**: Comprehensive end-to-end test validating the complete happy-path workflow from submission through execution completion, verifying all 8 MVP tasks work together.

**Phase**: [MVP]

## Test Scenario

**Given**: A valid workflow definition (research → verify → summarize) with real LangGraph executor and Redis storage

**When**: Client submits POST /execute with tenant_id, workflow_id, input

**Then**: Complete verification of:

1. Run creation (run_id, thread_id, status=queued)
2. Status transitions (queued → validating → pending → running → completed)
3. Node execution (node.started, node.completed events)
4. Event log accumulation (request.started, node.\*, request.completed)
5. State propagation (research_output, verification_output, summary_output)
6. Final status (completed) with all events
7. Event pagination (retrieve last 50 events)

## Test Flow

```
Step 1: Submit workflow via POST /execute
  - Assert 202 Accepted
  - Assert status=queued
  - Capture run_id, thread_id

Step 2: Poll GET /execute/{run_id}/status
  - Assert 200 OK
  - Assert status=pending or running (wait for transition)
  - Assert node.started event present

Step 3: Wait for execution
  - Poll until status=completed (max 30s)
  - Verify all node events emitted

Step 4: Verify final state
  - Assert status=completed
  - Assert research_output, verification_output, summary_output in state
  - Assert events: request.started, node.started x3, node.completed x3, request.completed

Step 5: Verify event log
  - Assert 18+ events in correct order
  - Assert all timestamps are ISO8601 UTC
  - Assert event.run_id matches submitted run_id

Step 6: Verify Redis persistence
  - Query Redis directly: run:{tenant_id}:{run_id}:* keys exist
  - Assert event count matches returned events
```

## Acceptance Criteria

- [ ] POST /execute returns 202 Accepted with status=queued
- [ ] Status transitions through all 5 states (queued, validating, pending, running, completed)
- [ ] All node events emitted in correct order
- [ ] Workflow state accumulates all node outputs
- [ ] Event log persists in Redis
- [ ] Events retrievable via GET /execute/{run_id}/status
- [ ] Test passes 100% of the time (deterministic)
- [ ] Test completes in <30 seconds

## Deliverables

1. `tests/test_e2e_workflow_execution.py` - New file (300+ lines)
   - `test_e2e_complete_workflow_submission_and_execution()` - main scenario
   - `test_e2e_event_log_accumulation()` - verify all events
   - `test_e2e_state_propagation()` - verify outputs accumulate
   - `test_e2e_concurrent_workflows()` - multiple runs simultaneously
   - `test_e2e_workflow_with_failed_node()` - error handling
   - `test_e2e_workflow_status_transitions_timing()` - verify transition timing

## Dependencies

- GW-MVP-RUNTIME-201: Status Lifecycle
- GW-MVP-RUNTIME-202: Real LangGraph Executor
- GW-MVP-RUNTIME-203: Event Emitter
- GW-MVP-DATA-201: Real Redis

## Environment Variables Required

| Variable                   | Purpose                                                  | Loaded By          |
| -------------------------- | -------------------------------------------------------- | ------------------ |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for status, events, and state persistence | Test fixture setup |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token                             | Test fixture setup |

**Reference**: See `[[../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading in Tests**:

```python
# In tests/test_e2e_workflow_execution.py
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
    # Cleanup
```

## Implementation Notes

- Use fixtures to set up test workflow (3-node research workflow)
- Use polling loop with backoff to wait for status transitions
- Verify Redis keys directly after completion (requires UPSTASH env vars)
- Use time.monotonic() to measure execution latency
- All assertions must include descriptive messages for debugging
- **All tests MUST have UPSTASH env vars loaded before execution**

## Test Coverage (6+ tests with comprehensive scenarios)

### Happy Path Tests

- [ ] `test_e2e_complete_workflow_submission_and_execution` - 3-node workflow start-to-finish
- [ ] `test_e2e_event_log_accumulation` - verify all 18+ events emitted correctly
- [ ] `test_e2e_state_propagation` - all node outputs accumulated in state
- [ ] `test_e2e_workflow_status_transitions_timing` - verify transition timing

### Concurrency and Scale Tests

- [ ] `test_e2e_concurrent_workflows` - multiple runs simultaneously (5+ parallel)
- [ ] `test_e2e_event_ordering_under_load` - event ordering preserved under concurrent execution
- [ ] `test_e2e_state_isolation_between_workflows` - verify no cross-contamination

### Error Handling Tests

- [ ] `test_e2e_workflow_with_failed_node` - error handling and recovery
- [ ] `test_e2e_node_timeout_halts_execution` - timeout enforcement
- [ ] `test_e2e_malformed_edge_condition` - invalid JSONPath handling

### Timing and Latency Tests

- [ ] All status transitions complete within SLA (<100ms per transition)
- [ ] Event retrieval <500ms for full log
- [ ] Execution latency <30s for 3-node workflow

### Testing Strategy

- Happy path: Full workflow execution end-to-end with real Redis (all UPSTASH env vars loaded)
- Concurrency: 5+ parallel workflows, verify no interference
- Event ordering: Verify request.started, node.started x3, node.completed x3, request.completed in order
- State propagation: All node outputs present in final state
- Error scenarios: Failed nodes, timeouts, malformed conditions
- Performance: Measure execution latency, verify <30s for typical workflow
- **All tests MUST have UPSTASH env vars loaded before execution**
- Redis cleanup: Event logs and checkpoints removed after test completion
