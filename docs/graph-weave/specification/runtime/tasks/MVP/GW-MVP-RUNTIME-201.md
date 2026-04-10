# GW-MVP-RUNTIME-201: Real Execution Status Lifecycle

**Objective**: Implement the 7-state execution status lifecycle (queued, validating, pending, running, completed, failed, cancelled) with real state transitions in Redis instead of mocked in-memory state.

**Phase**: [MVP]

## Requirements

### Functional

- Status enum must enforce 7 states: queued, validating, pending, running, completed, failed, cancelled
- POST /execute must return 202 Accepted with status=queued immediately (not 201 Created with pending)
- Background job must transition: queued → validating → pending → running → {completed|failed|cancelled}
- GET /execute/{run_id}/status must return 200 OK with current status and event log for valid runs
- GET /execute/{run_id}/status must return 404 Not Found for non-existent run_ids
- Status must be persisted in Redis under key: run:{tenant_id}:{run_id}:status
- Status transitions must be atomic (use Redis transactions or lua scripts to prevent race conditions)
- Run creation must store: run_id, thread_id, workflow_id, tenant_id, created_at, initial_status=queued

### Non-Functional

- Status transitions must complete within 100ms (prevent blocking on background job transitions)
- No status transition may leave run in inconsistent state
- Status updates must be auditable (emit status.changed events for each transition)

## Implementation Approach

1. Replace in-memory status storage with Redis-backed storage
2. Create `services/status_service.py` with:
   - `get_status(run_id, tenant_id)` → fetch from Redis
   - `set_status(run_id, tenant_id, new_status)` → atomic update with validation
   - `transition_status(run_id, tenant_id, current_status, target_status)` → validate transition, emit event
3. Update `main.py` POST /execute endpoint to:
   - Create run_id and thread_id
   - Store run metadata in Redis with status=queued
   - Return 202 Accepted immediately (don't wait for background job)
4. Create background job (`background_jobs/status_transitions.py`) to:
   - Poll for queued runs
   - Transition queued → validating → pending → running
   - Let executor handle running → {completed|failed}
5. Update GET /execute/{run_id}/status to fetch from Redis and return 404 if not found
6. Add status.changed event emission in transition function

## Acceptance Criteria

- [ ] StatusEnum defined with 7 states (queued, validating, pending, running, completed, failed, cancelled)
- [ ] POST /execute returns 202 Accepted with status=queued (verify via test)
- [ ] Run data persists in Redis with run:{tenant_id}:{run_id}:\* keys
- [ ] GET /execute/{run_id}/status returns 200 with status for valid run_ids
- [ ] GET /execute/{run_id}/status returns 404 for invalid run_ids
- [ ] Status transitions are atomic (no partial updates visible to concurrent requests)
- [ ] All tests passing (expect 210+ from new status tests)
- [ ] lsp_diagnostics clean on modified files

## Related Requirements

- FR-RUNTIME-002 [MOCK,MVP]: Workflow submission must return a run id for later status access
- FR-RUNTIME-004 [MVP,FULL]: Checkpoints and active thread state must survive interruptions and completion

## Deliverables

1. `src/models.py` - Update ExecuteResponse to use StatusEnum instead of string
2. `src/services/status_service.py` - New service for Redis-backed status management (120 lines)
3. `src/background_jobs/status_transitions.py` - New background job (100 lines)
4. `src/main.py` - Update POST /execute to use new status service, return 202
5. `tests/test_status_lifecycle.py` - New test file (150 lines, 12+ test cases)

## Implementation Notes

- Decision 2 from MVP spec: Status lifecycle includes intermediate states (queued, validating, pending)
- Decision 1: HTTP status 404 for non-existent runs ensures clients know run doesn't exist
- Redis key format: run:{tenant_id}:{run_id}:status stores current status as string
- Background job should use exponential backoff to avoid hammering Redis
- Status transitions must emit events (ties to GW-MVP-RUNTIME-203 Event Emitter)
- This task is prerequisite for checkpoint recovery (GW-MVP-DATA-203)

## Environment Variables Required

| Variable                   | Purpose                               | Loaded By                                      |
| -------------------------- | ------------------------------------- | ---------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for status persistence | `src/config.py` → `services/status_service.py` |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token          | `src/config.py` → `services/status_service.py` |

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading**:

```python
# In services/status_service.py
from src.config import Config

class StatusService:
    def __init__(self):
        self.redis_client = RedisClient(
            url=Config.UPSTASH_REDIS_REST_URL,
            token=Config.UPSTASH_REDIS_REST_TOKEN
        )
```

## Test Coverage (15+ tests)

### Unit Tests (no Redis needed)

- [ ] StatusEnum has 7 states (queued, validating, pending, running, completed, failed, cancelled)
- [ ] Status transitions reject invalid transitions (e.g., completed → queued)
- [ ] Status transitions allow valid paths (queued → validating → pending → running → completed)
- [ ] Status transitions allow cancel from running (running → cancelled)
- [ ] Status transitions allow fail from running (running → failed)
- [ ] Run metadata schema: run_id, thread_id, workflow_id, tenant_id, created_at, status

### Integration Tests (with Redis)

- [ ] POST /execute creates run with status=queued, verify in Redis
- [ ] GET /execute/{run_id}/status returns 200 with status for valid run
- [ ] GET /execute/{run_id}/status returns 404 for non-existent run
- [ ] Status transitions atomic (no partial updates visible)
- [ ] Status updated in Redis with correct TTL
- [ ] Run metadata persists in Redis under run:{tenant_id}:{run_id}:\* keys

### Concurrency Tests

- [ ] Multiple threads transition same run status without race conditions
- [ ] Concurrent status checks return consistent values
- [ ] Status update and read atomic (no torn reads)

### Error Scenarios

- [ ] Invalid status string → raises ValueError
- [ ] Transition to same status idempotent
- [ ] Redis unavailable → graceful error handling
- [ ] Concurrent transition attempts → only one succeeds

### Timing Tests

- [ ] Status transition <100ms (non-blocking)
- [ ] GET /execute/{run_id}/status responds <500ms

## Testing Strategy

- Unit: StatusEnum validates state machine (queued → validating → pending → running → completed)
- Unit: Status transitions reject invalid transitions (e.g., completed → queued)
- Integration: POST /execute creates run with status=queued, verify in Redis (requires UPSTASH env vars loaded)
- Integration: GET /execute/{run_id}/status returns 200 with status for valid run, 404 for invalid (requires UPSTASH env vars)
- Concurrency: Multiple threads updating same run status, verify no race conditions
- Error scenarios: Redis unavailable, invalid transitions, concurrent updates
- E2E: Full workflow submission → status tracking → completion (covered in GW-MVP-E2E-001, requires all env vars)
