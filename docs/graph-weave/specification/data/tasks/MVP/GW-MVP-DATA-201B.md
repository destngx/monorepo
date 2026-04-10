# GW-MVP-DATA-201B: Redis Namespacing & Circuit Breaker

**Objective**: Implement tenant-scoped key namespacing and circuit breaker pattern for Redis reliability and fallback storage.

**Phase**: [MVP]

**Duration**: 1.5 hours

**Blocked By**: DATA-201A

**Blocks**: RUNTIME-201, RUNTIME-202, RUNTIME-203, DATA-202, DATA-203, E2E-001, E2E-002, E2E-003

## Requirements

### Functional

- Tenant-scoped key format: `{type}:{tenant_id}:{resource_id}`
- Key types: `run:{tenant_id}:{run_id}`, `workflow:{tenant_id}:{workflow_id}:{version}`, `thread:{tenant_id}:{thread_id}`
- TTL by key type:
  - runs: 7 days
  - workflows: no expiration
  - threads: 1 hour
  - checkpoints: 30 days
  - events: no expiration (pruned via LTRIM)
- Circuit breaker pattern:
  - States: CLOSED (normal), OPEN (failing), HALF_OPEN (retrying)
  - Failure threshold: 3 consecutive failures
  - Backoff: 100ms exponential
  - Fallback storage: in-memory dict when Redis unavailable
- Fallback storage: LRU eviction at 1000 keys
- Health check: GET /health returns `{"status": "healthy"/"degraded", "redis": bool}`
- Tenant isolation: operations scoped to tenant, Tenant A cannot read Tenant B keys

### Non-Functional

- Namespace helpers generate correct key format instantly
- TTL enforcement automatic
- Circuit breaker transparent (same API as normal Redis)
- Fallback storage limited to 1000 keys (LRU eviction)
- State machine transitions atomic (no partial state)

## Implementation Approach

1. Create `src/adapters/redis_circuit_breaker.py`:
   - `NamespacedRedisClient` class wrapping `RedisClient`:
     - `run_key(run_id, tenant_id)` → `"run:{tenant_id}:{run_id}"`
     - `workflow_key(workflow_id, tenant_id, version)` → `"workflow:{tenant_id}:{workflow_id}:{version}"`
     - `thread_key(thread_id, tenant_id)` → `"thread:{tenant_id}:{thread_id}"`
     - `set_with_ttl(key_type, value, tenant_id, resource_id, **kwargs)`
   - `CircuitBreaker` class:
     - State machine: CLOSED → OPEN → HALF_OPEN → CLOSED
     - Failure counter and backoff timer
     - Transparent error handling
   - `FallbackStorage` class:
     - In-memory dict with LRU eviction
     - Same interface as RedisClient
   - Health check: queries Redis status, returns degraded if unavailable

2. TTL configuration:

   ```python
   TTL_CONFIG = {
     "run": 7 * 24 * 3600,        # 7 days
     "workflow": None,             # no expire
     "thread": 1 * 3600,           # 1 hour
     "checkpoint": 30 * 24 * 3600, # 30 days
     "event": None                 # no expire (pruned via LTRIM)
   }
   ```

3. State machine diagram:
   ```
   CLOSED (normal) --(3 failures)--> OPEN (failing)
                                         |
                                    wait 100ms
                                         |
   HALF_OPEN (retrying) <----------------+
      |
      +-(success)--> CLOSED
      +-(failure)--> OPEN
   ```

## Acceptance Criteria

- [ ] NamespacedRedisClient generates correct tenant-scoped keys
- [ ] Tenant A cannot read Tenant B keys (isolation verified)
- [ ] TTL applied correctly per key type
- [ ] Circuit breaker CLOSED state: normal Redis operations
- [ ] Circuit breaker OPEN state: operations fail immediately
- [ ] Circuit breaker HALF_OPEN state: single retry attempted
- [ ] Circuit breaker recovers to CLOSED after retry succeeds
- [ ] Fallback storage active when Redis unavailable
- [ ] Fallback storage evicts LRU at 1001 keys
- [ ] Health check endpoint returns accurate status
- [ ] All tests passing (20+ tests)
- [ ] lsp_diagnostics clean
- [ ] No race conditions under concurrent access

## Related Requirements

- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant isolation must apply to Redis state
- FR-DATA-002 [MVP]: Circuit breaker for Redis availability
- FR-RUNTIME-030 [MVP]: Health check endpoint

## Deliverables

1. `src/adapters/redis_circuit_breaker.py` (120 LOC)
2. `tests/test_redis_circuit_breaker.py` (200+ LOC, 20+ tests)

## Test Coverage (20+ tests)

### Unit Tests (no Redis needed)

- [ ] run_key generates `"run:{tenant_id}:{run_id}"`
- [ ] workflow_key generates `"workflow:{tenant_id}:{workflow_id}:{version}"`
- [ ] thread_key generates `"thread:{tenant_id}:{thread_id}"`
- [ ] CircuitBreaker initializes in CLOSED state
- [ ] CLOSED → OPEN after 3 consecutive failures
- [ ] OPEN → HALF_OPEN after backoff expires
- [ ] HALF_OPEN → CLOSED on successful operation
- [ ] HALF_OPEN → OPEN on failed operation
- [ ] Fallback storage stores keys when Redis unavailable
- [ ] Fallback storage evicts LRU when exceeds 1000 keys
- [ ] Tenant isolation: run_key differs for different tenants
- [ ] Circuit breaker state transitions atomic

### Integration Tests (with real Upstash instance)

- [ ] SET with TTL: verify Redis TTL correct for run key (7 days)
- [ ] SET with TTL: verify Redis TTL correct for thread key (1 hour)
- [ ] SET with TTL: verify workflow key has no expiration
- [ ] Circuit breaker transparent: normal operation in CLOSED
- [ ] Circuit breaker fallback: operations succeed in OPEN (in-memory)
- [ ] Circuit breaker recovery: transitions CLOSED after backoff
- [ ] Health check returns `healthy=true` when Redis available
- [ ] Health check returns `healthy=false` when Redis unavailable
- [ ] Tenant isolation test: set key for tenant_1, verify tenant_2 cannot read

## Concurrency Tests

- [ ] Multiple threads concurrently access Redis through circuit breaker
- [ ] Race condition: verify no lost updates during state transitions
- [ ] Race condition: verify no partial state when switching OPEN/CLOSED
- [ ] Concurrent fallback access: no data corruption

## Error Scenarios

- [ ] Circuit breaker: all 3 retries fail → OPEN state
- [ ] Circuit breaker: backoff expires → HALF_OPEN, retry succeeds
- [ ] Redis unavailable → fallback storage used
- [ ] Fallback storage full (1001 keys) → LRU eviction happens
- [ ] Recover from fallback: Redis comes back online, data synced

## Environment Variables

- `UPSTASH_REDIS_REST_URL` (required)
- `UPSTASH_REDIS_REST_TOKEN` (required)

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy.

## Implementation Notes

- Use `time.monotonic()` for backoff timing (not wall clock)
- Implement state machine carefully (consider using enum)
- LRU eviction: use `collections.OrderedDict` for O(1) eviction
- All state transitions must be atomic (consider using locks)
- This task depends on DATA-201A (RedisClient must work)
- All other tasks wait for this to complete before proceeding

## Testing Strategy

- Unit: State machine transitions, key formatting, LRU eviction
- Integration: Real Upstash instance with circuit breaker
- Concurrency: Multiple threads under load
- Error scenarios: Network failures, timeouts, fallback activation
- Recovery: Data consistency after fallback and recovery
