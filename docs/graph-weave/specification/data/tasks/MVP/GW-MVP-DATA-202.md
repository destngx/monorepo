# GW-MVP-DATA-202: Checkpoint Persistence and Recovery

**Objective**: Implement checkpoint persistence in Redis for thread recovery and interruption handling, allowing resumed execution to restore state from last checkpoint.

**Phase**: [MVP]

## Requirements

### Functional

- Checkpoint must capture: node_id, workflow_state snapshot, timestamp, thread_id
- Checkpoints must be written after each node execution (not blocking)
- Recovery must restore workflow_state from last checkpoint and resume at next node
- Checkpoint key format: checkpoint:{tenant_id}:{thread_id}:latest
- Recovery endpoint: POST /execute/{run_id}/recover with thread_id in payload
- Thread cleanup must remove checkpoint on completion
- Checkpoint TTL: 30 days (allow recovery window for long-running workflows)

### Non-Functional

- Checkpoint write must not block execution (<10ms)
- Checkpoint size must not exceed 1MB per thread
- Recovery latency <1s (restore from Redis)

## Implementation Approach

1. Create `services/checkpoint_service.py` with:
   - `save_checkpoint(run_id, thread_id, tenant_id, workflow_state)` → write to Redis
   - `load_checkpoint(thread_id, tenant_id)` → fetch from Redis
   - `clear_checkpoint(thread_id, tenant_id)` → delete on completion
2. Update executor to call save_checkpoint() after each node
3. Create recovery endpoint: POST /execute/{run_id}/recover
4. Add recovery route handler that loads checkpoint and resumes execution

## Acceptance Criteria

- [ ] Checkpoint saved after each node execution
- [ ] Checkpoint persists in Redis with correct TTL
- [ ] Recovery endpoint loads checkpoint and restores state
- [ ] Resumed execution continues from correct node
- [ ] Checkpoint cleanup on workflow completion
- [ ] Tests passing (60+ new tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-004 [MVP,FULL]: Checkpoints and active thread state must survive interruptions and completion

## Deliverables

1. `src/services/checkpoint_service.py` - New checkpoint service (80 lines)
2. `src/main.py` - Add POST /execute/{run_id}/recover endpoint (40 lines)
3. `tests/test_checkpoint_recovery.py` - Recovery tests (150 lines, 20+ tests)

## Environment Variables Required

| Variable                   | Purpose                               | Loaded By                                          |
| -------------------------- | ------------------------------------- | -------------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for checkpoint storage | `src/config.py` → `services/checkpoint_service.py` |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token          | `src/config.py` → `services/checkpoint_service.py` |

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading**:

```python
# In services/checkpoint_service.py
from src.config import Config

class CheckpointService:
    def __init__(self):
        self.redis_client = RedisClient(
            url=Config.UPSTASH_REDIS_REST_URL,
            token=Config.UPSTASH_REDIS_REST_TOKEN
        )
```

## Test Coverage (20+ tests)

### Unit Tests (no Redis needed)

- [ ] Checkpoint schema validation: requires node_id, workflow_state, timestamp, thread_id
- [ ] Checkpoint timestamp generation produces ISO8601 UTC
- [ ] Checkpoint JSON serialization roundtrip preserves state
- [ ] Workflow_state dict structure preserved in checkpoint

### Integration Tests (with Redis)

- [ ] save_checkpoint() stores in Redis with correct key format
- [ ] save_checkpoint() respects 30-day TTL
- [ ] load_checkpoint() retrieves saved state correctly
- [ ] Restored state identical to saved state
- [ ] Multiple checkpoints per thread (overwrite latest)
- [ ] clear_checkpoint() deletes checkpoint completely
- [ ] Checkpoint persists across app restart
- [ ] POST /execute/{run_id}/recover loads checkpoint
- [ ] Resumed execution continues from correct node

### Concurrency Tests

- [ ] Multiple threads saving checkpoints concurrently
- [ ] Checkpoint save doesn't block execution (async OK)
- [ ] Multiple recovery attempts race-condition safe
- [ ] Concurrent loads retrieve consistent state

### Error Scenarios

- [ ] Non-existent checkpoint → load_checkpoint() returns None
- [ ] Save oversized state (>1MB) → rejected or truncated
- [ ] Redis unavailable → graceful error handling
- [ ] Malformed checkpoint data → deserialization error caught
- [ ] Recovery with invalid thread_id → 404 response
- [ ] Save during concurrent write → last write wins

### State Consistency Tests

- [ ] Checkpoint captures all accumulated node outputs
- [ ] State restored with all fields present
- [ ] Resumed execution produces same output as continuous

## Testing Strategy

- Unit: Checkpoint schema validation (no Redis needed)
- Integration: Checkpoint saved and retrieved from Redis with UPSTASH env vars loaded
- Integration: State correctly restored and execution resumes with UPSTASH env vars loaded
- Concurrency: Multiple threads saving checkpoints, verify no data loss
- Error scenarios: Missing checkpoints, corrupted data, Redis failures
- State consistency: Interrupted+recovered output matches non-interrupted
- E2E: Interrupt workflow, recover, verify continued execution (GW-MVP-E2E-003, all env vars required)
