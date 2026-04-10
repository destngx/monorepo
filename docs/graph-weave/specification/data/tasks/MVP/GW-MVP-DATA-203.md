# GW-MVP-DATA-203: Active Thread Lifecycle and Cleanup

**Objective**: Implement active thread tracking and cleanup to manage thread lifecycle from creation through completion, enabling cancellation and preventing orphaned threads.

**Phase**: [MVP]

## Requirements

### Functional

- Active threads tracked in Redis set: active_threads:{tenant_id}
- Thread lifecycle: queued → validating → pending → running → {completed|failed|cancelled}
- On completion: remove from active_threads set, clear checkpoint, clear execution state
- Cancellation endpoint: POST /execute/{run_id}/cancel to interrupt thread
- Cancel must set kill flag: circuit_breaker:{tenant_id}:{thread_id}:kill
- Thread cleanup must be deterministic (no orphaned threads)
- List active threads: GET /threads (admin endpoint, list by tenant)

### Non-Functional

- Cleanup must complete within 100ms
- No orphaned threads after 1 hour of inactivity
- Memory: active_threads set should not exceed 100KB per 1000 concurrent threads

## Implementation Approach

1. Create `services/thread_lifecycle_service.py` with:
   - `add_active_thread(thread_id, tenant_id)` → add to set
   - `remove_active_thread(thread_id, tenant_id)` → remove from set
   - `is_active(thread_id, tenant_id)` → check membership
   - `list_active_threads(tenant_id)` → enumerate
2. Create kill switch service: `set_kill_flag(thread_id, tenant_id)` → store kill flag in Redis
3. Executor checks kill flag after each node (circuit breaker pattern)
4. Add POST /execute/{run_id}/cancel endpoint
5. Add cleanup job: periodic scan for stale threads (>1h), remove

## Acceptance Criteria

- [ ] Active threads tracked in Redis set
- [ ] Threads added on creation, removed on completion
- [ ] Cancellation sets kill flag, executor respects it
- [ ] Stale thread cleanup working
- [ ] Tests passing (50+ new tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-004 [MVP,FULL]: Checkpoints and active thread state must survive interruptions and completion

## Deliverables

1. `src/services/thread_lifecycle_service.py` - Thread tracking (100 lines)
2. `src/main.py` - Add POST /execute/{run_id}/cancel, GET /threads endpoints
3. `tests/test_thread_lifecycle.py` - Lifecycle tests (120 lines, 15+ tests)

## Environment Variables Required

| Variable                   | Purpose                                           | Loaded By                                                |
| -------------------------- | ------------------------------------------------- | -------------------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for thread tracking and kill flags | `src/config.py` → `services/thread_lifecycle_service.py` |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token                      | `src/config.py` → `services/thread_lifecycle_service.py` |

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading**:

```python
# In services/thread_lifecycle_service.py
from src.config import Config

class ThreadLifecycleService:
    def __init__(self):
        self.redis_client = RedisClient(
            url=Config.UPSTASH_REDIS_REST_URL,
            token=Config.UPSTASH_REDIS_REST_TOKEN
        )
```

## Test Coverage (20+ tests)

### Unit Tests (no Redis needed)

- [ ] Thread lifecycle states: queued, validating, pending, running, completed, failed, cancelled
- [ ] Valid state transitions enforced (no backwards transitions)
- [ ] Thread metadata schema: thread_id, tenant_id, workflow_id, status, created_at

### Integration Tests (with Redis)

- [ ] add_active_thread() adds thread to active_threads set in Redis
- [ ] is_active() returns True/False correctly
- [ ] remove_active_thread() removes from set
- [ ] list_active_threads() enumerates all active threads
- [ ] Active threads set persists in Redis
- [ ] Multiple threads tracked independently
- [ ] Stale thread cleanup working (1h timeout)

### Concurrency Tests

- [ ] Multiple threads added concurrently without data loss
- [ ] Active thread count accurate under concurrent add/remove
- [ ] Concurrent removal attempts race-condition safe
- [ ] List active threads reads consistent snapshot

### Cancellation Tests

- [ ] set_kill_flag() stores flag in Redis
- [ ] Executor checks kill flag after each node
- [ ] Kill flag causes immediate halt
- [ ] Kill flag prevents node execution

### Error Scenarios

- [ ] Non-existent thread → is_active() returns False
- [ ] Remove non-existent thread → no error (idempotent)
- [ ] Redis unavailable → graceful error handling
- [ ] Concurrent add/remove same thread → consistent state
- [ ] Kill flag set to non-existent thread → stored anyway (idempotent)

### Cleanup Tests

- [ ] Completed threads removed from active_threads
- [ ] Checkpoint cleared on completion
- [ ] All state cleaned up after thread finishes
- [ ] Stale thread cleanup removes >1h inactive threads

## Testing Strategy

- Unit: Thread state transitions valid (no Redis needed)
- Integration: Threads added/removed from active set correctly with UPSTASH env vars loaded
- Integration: Cancellation kills execution with UPSTASH env vars loaded
- Concurrency: Multiple threads modifying active set, verify no data loss
- Error scenarios: Missing threads, concurrent updates, redis failures
- Cleanup: Stale threads removed, state cleaned up
- E2E: Full thread lifecycle (create → run → complete → cleanup) with all env vars required
