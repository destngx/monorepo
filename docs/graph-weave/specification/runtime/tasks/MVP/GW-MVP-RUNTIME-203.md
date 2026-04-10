# GW-MVP-RUNTIME-203: Event Emitter with Real Event Log

**Objective**: Implement real event emission and storage in Redis event log, replacing mock in-memory events with persistent, ordered, immutable event streams.

**Phase**: [MVP]

## Requirements

### Functional

- Event log must be append-only (immutable once written)
- Events must include: type, run_id, timestamp (ISO8601 UTC), data
- 18 event types must be supported: request._, node._, tool._, input._, state._, checkpoint._, complete
- Events must be ordered by timestamp within a run
- GET /execute/{run_id}/status must return paginated event log (default: last 50 events)
- Event storage must use Redis RPUSH (append-only list) under key: run:{tenant_id}:{run_id}:events
- Timestamp must be server-generated (not client-supplied)
- Events must be queryable by type (future enhancement marked [FULL])

### Non-Functional

- Event emission must not block execution (<1ms per emit)
- Event log persistence must not fail workflow (best-effort, errors logged)
- Memory: event log should not exceed 1MB per run (enforce via LTRIM on old events)

## Implementation Approach

1. Create `services/event_emitter.py` with:
   - `emit_event(run_id, tenant_id, event_type, data)` → append to Redis + return event
   - `get_events(run_id, tenant_id, limit=50, offset=0)` → fetch from Redis RPUSH list
   - `clear_events(run_id, tenant_id)` → delete event log (for cleanup)
2. Event types must be enum: EventType with 18 values (request.started, node.completed, etc.)
3. Event schema: `{type, run_id, timestamp, data}`
4. Update executor to call emit_event() for each milestone
5. Update GET /execute/{run_id}/status to fetch events and return in response
6. Add optional pagination: `?offset=0&limit=50` query parameters

## Acceptance Criteria

- [ ] EventType enum with 18 event types defined
- [ ] emit_event() function appends to Redis with proper key format
- [ ] get_events() retrieves from Redis and returns paginated list
- [ ] Events include proper timestamps (ISO8601 UTC)
- [ ] All executor milestones emit events (node.started, node.completed, etc.)
- [ ] GET /execute/{run_id}/status returns events in response
- [ ] Event log persists across app restarts
- [ ] Tests passing (80+ new tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-RUNTIME-003 [MOCK,MVP,FULL]: A separate SSE status request must stream structured events for the submitted run

## Deliverables

1. `src/models.py` - Add EventType enum (18 values), Event schema
2. `src/services/event_emitter.py` - New event service (100 lines)
3. `tests/test_event_emitter.py` - Unit tests for event emission (150 lines, 20+ tests)

## Implementation Notes

- Decision 4: 18 event types locked in spec (Section 3.5 request-lifecycle.md)
- Events must be immutable once written (no updates, only appends)
- Redis memory management: use LTRIM to keep only last 1000 events per run
- Timestamp generation: use `datetime.now(timezone.utc).isoformat() + "Z"`
- This task depends on GW-MVP-DATA-201 (Real Redis)
- Ties to GW-MVP-E2E-001 for end-to-end event tracking verification

## Environment Variables Required

| Variable                   | Purpose                              | Loaded By                                     |
| -------------------------- | ------------------------------------ | --------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | Redis endpoint for event log storage | `src/config.py` → `services/event_emitter.py` |
| `UPSTASH_REDIS_REST_TOKEN` | Upstash authentication token         | `src/config.py` → `services/event_emitter.py` |

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy and validation rules.

**Configuration Loading**:

```python
# In services/event_emitter.py
from src.config import Config

class EventEmitter:
    def __init__(self):
        self.redis_client = RedisClient(
            url=Config.UPSTASH_REDIS_REST_URL,
            token=Config.UPSTASH_REDIS_REST_TOKEN
        )
```

## Test Coverage (20+ tests)

### Unit Tests (no Redis needed)

- [ ] EventType enum has 18 event types correctly named
- [ ] Event schema has required fields: type, run_id, timestamp, data
- [ ] Timestamp generation produces ISO8601 UTC format
- [ ] Event creation from dict validates required fields
- [ ] Event JSON serialization roundtrip preserves data

### Integration Tests (with Redis)

- [ ] emit_event() correctly stores in Redis with RPUSH
- [ ] emit_event() generates correct timestamp (ISO8601 UTC)
- [ ] get_events() retrieves from Redis in order
- [ ] get_events() pagination works (limit, offset)
- [ ] get_events() returns last 50 by default
- [ ] Multiple events accumulate correctly
- [ ] Event log survives app restart (persisted in Redis)
- [ ] clear_events() deletes event log completely

### Concurrency Tests

- [ ] Multiple threads emit events concurrently without data loss
- [ ] Event ordering preserved under concurrent emission
- [ ] No duplicate events with concurrent emitters
- [ ] Event count accurate with 100+ concurrent emits

### Error Scenarios

- [ ] Invalid event type → raises ValueError
- [ ] Emit to non-existent run → still stores (creates new log)
- [ ] Redis unavailable → graceful error handling
- [ ] Oversized event data → truncated or rejected gracefully
- [ ] Malformed timestamp → regenerated by server

### Memory and Cleanup Tests

- [ ] Event log LTRIM keeps only last 1000 events per run
- [ ] Old events removed when log exceeds 1000
- [ ] Event storage does not exceed 1MB per run

## Testing Strategy

- Unit: EventType enum has 18 values, all correctly named (no Redis needed)
- Unit: Event schema validates required fields (type, run_id, timestamp, data)
- Unit: Timestamps are ISO8601 UTC format
- Integration: emit_event() correctly stores in Redis with UPSTASH env vars loaded
- Integration: get_events() retrieves from Redis in order with UPSTASH env vars loaded
- Concurrency: Multiple threads emit events, verify no data loss and ordering preserved
- Error scenarios: Invalid events, missing fields, malformed data
- Cleanup: LTRIM enforced at 1000 events per run
- E2E: Full execution emits all expected events in order (in GW-MVP-E2E-001, all env vars required)
