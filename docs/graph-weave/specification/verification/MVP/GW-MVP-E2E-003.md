# GW-MVP-E2E-003: End-to-End Checkpoint Recovery and Thread Cancellation

**Objective**: Verify checkpoint persistence and recovery work end-to-end, allowing interrupted workflows to resume from last checkpoint, and verify cancellation properly terminates execution.

**Phase**: [MVP]

## Test Scenario A: Checkpoint and Recovery

**Given**:

- Workflow with 5-node execution path
- Real checkpoint service persisting to Redis
- Recovery endpoint enabled

**When**:

1. Workflow starts execution
2. Completes node 1-3 (saves checkpoints)
3. Execution interrupted (kill flag set)
4. Recovery endpoint called with thread_id

**Then**: Verify:

1. Checkpoint saved after each node (4 checkpoints for nodes 1-4)
2. On recovery: state restored to last checkpoint
3. Execution resumes from node 4 (not from beginning)
4. Final output same as if execution never interrupted
5. All events preserved in log (before interrupt + recovery events)

## Test Flow (Checkpoint)

```
Step 1: Submit 5-node workflow
Step 2: Poll until node.2 or node.3 execution (capture thread_id)
Step 3: Simulate interrupt (set kill flag in Redis)
Step 4: Verify execution halts (circuit breaker stops)
Step 5: Verify checkpoint exists in Redis
Step 6: Call POST /execute/{run_id}/recover with thread_id
Step 7: Poll until execution completes
Step 8: Verify final output is complete
Step 9: Verify event log shows both original nodes + recovery
```

## Test Scenario B: Cancellation

**Given**:

- Workflow with long-running agent node (>5s)
- Cancellation endpoint enabled
- Active thread tracking

**When**:

1. Workflow submitted and starts execution
2. Client calls POST /execute/{run_id}/cancel
3. Circuit breaker detects kill flag

**Then**: Verify:

1. Thread removed from active_threads set
2. Kill flag prevents node execution
3. Execution halts cleanly
4. Status transitions to cancelled
5. Checkpoint removed (cleanup)
6. Final event is request.cancelled

## Test Flow (Cancellation)

```
Step 1: Submit workflow with long-running node
Step 2: Poll until status=running (node executing)
Step 3: Verify thread_id in active_threads set
Step 4: Call POST /execute/{run_id}/cancel
Step 5: Verify kill flag set in Redis
Step 6: Poll until status=cancelled
Step 7: Verify thread removed from active_threads
Step 8: Verify request.cancelled event emitted
Step 9: Verify checkpoint cleared
```

## Acceptance Criteria

- [ ] Checkpoint saved after each node in Redis
- [ ] Recovery loads checkpoint and restores state correctly
- [ ] Resume execution continues from right node (not restart)
- [ ] Final output matches non-interrupted execution
- [ ] Event log includes both original + recovery events
- [ ] Cancellation sets kill flag and halts execution
- [ ] Status transitions to cancelled on abort
- [ ] Active threads cleaned up
- [ ] All checkpoints cleaned on completion
- [ ] Tests pass with multiple interruption scenarios

## Deliverables

1. `tests/test_e2e_checkpoint_recovery.py` - New file (300+ lines)
   - `test_e2e_checkpoint_after_each_node()` - verify checkpointing
   - `test_e2e_recovery_restores_state()` - recovery verification
   - `test_e2e_resume_execution_continues()` - continuation test
   - `test_e2e_interrupted_then_recovered_matches_normal()` - equivalence test
   - `test_e2e_multiple_interruptions_and_recoveries()` - stress test
   - `test_e2e_cancellation_halts_execution()` - cancellation test
   - `test_e2e_cancel_removes_active_thread()` - thread cleanup test
   - `test_e2e_cancel_emits_cancelled_event()` - event verification test

## Dependencies

- GW-MVP-DATA-202: Checkpoint Persistence
- GW-MVP-DATA-203: Active Thread Lifecycle
- GW-MVP-RUNTIME-201: Status Lifecycle (for cancelled status)

## Environment Variables Required

| Variable                   | Purpose                                                                       | Loaded By          |
| -------------------------- | ----------------------------------------------------------------------------- | ------------------ |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for checkpoint storage, kill flags, and active thread tracking | Test fixture setup |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token                                                  | Test fixture setup |

**Reference**: See `[[../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading in Tests**:

```python
# In tests/test_e2e_checkpoint_recovery.py
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

## Test Coverage (8+ tests for checkpoint recovery and cancellation)

### Checkpoint and Recovery Tests

- [ ] `test_e2e_checkpoint_after_each_node` - verify checkpoint saved after nodes 1-4
- [ ] `test_e2e_recovery_restores_state` - recovery loads checkpoint correctly
- [ ] `test_e2e_resume_execution_continues` - execution continues from node 4 (not restart)
- [ ] `test_e2e_interrupted_then_recovered_matches_normal` - output identical to non-interrupted
- [ ] `test_e2e_multiple_interruptions_and_recoveries` - interrupt at node 2, 4, recover both

### Checkpoint Stress Tests

- [ ] `test_e2e_checkpoint_with_large_state` - checkpoint stores large workflow_state
- [ ] `test_e2e_concurrent_checkpoints` - multiple threads saving checkpoints simultaneously

### Cancellation Tests

- [ ] `test_e2e_cancellation_halts_execution` - kill flag stops workflow immediately
- [ ] `test_e2e_cancel_removes_active_thread` - thread removed from active_threads set
- [ ] `test_e2e_cancel_emits_cancelled_event` - request.cancelled event emitted
- [ ] `test_e2e_cancel_cleans_checkpoint` - checkpoint deleted on cancellation
- [ ] `test_e2e_cancel_cleans_state` - all execution state cleaned up

### Edge Cases and Race Conditions

- [ ] `test_e2e_recovery_after_multiple_cancellations` - recover after cancel attempt
- [ ] `test_e2e_concurrent_recovery_attempts` - multiple recovery requests race-condition safe
- [ ] `test_e2e_cancel_during_checkpoint_save` - cancel while saving checkpoint
- [ ] `test_e2e_interrupt_at_node_boundaries` - interrupt exactly at node transition

### Timing and Cleanup Tests

- [ ] Recovery completes <1s (checkpoint loading performance)
- [ ] Cancellation takes effect <500ms
- [ ] All checkpoints cleaned after completion (no orphaned data)
- [ ] Event logs remain accessible after cancellation

## Implementation Notes

- Use time.sleep() to simulate long-running nodes
- Set kill flag manually in Redis for interrupt simulation (requires UPSTASH env vars)
- Verify checkpoint schema in Redis directly (requires UPSTASH env vars)
- Use fixtures for multi-node workflows
- Assert event order (original events before recovery events)
- Verify state equivalence (interrupted+recovered == direct execution)
- **All tests MUST have UPSTASH env vars loaded before execution**
- Redis cleanup (kill flags, checkpoints, active_threads) verified after each test
- Concurrency: Multiple concurrent recoveries should not interfere
- Race conditions: Cancel and recovery attempted simultaneously should be safe
