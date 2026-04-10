# GW-MVP-DATA-201A: Redis Connection & Adapter

**Objective**: Implement HTTP-based Redis client via Upstash REST API with connection pooling, retry logic, and foundational operations for all downstream tasks.

**Phase**: [MVP]

**Duration**: 1.5 hours

**Blocked By**: NONE (foundational - can start immediately)

**Blocks**: DATA-201B, RUNTIME-201, RUNTIME-202, RUNTIME-203, DATA-202, DATA-203, E2E-001, E2E-002, E2E-003

## Requirements

### Functional

- HTTP-based Upstash Redis client (not redis-py)
- Core operations: GET, SET, DELETE, RPUSH, LPUSH, LRANGE, LTRIM, TTL, MGET, EXISTS
- Retry logic: exponential backoff (100ms initial, max 3 retries)
- Error handling: timeout, connection refused, 5xx responses
- Request/response serialization (JSON)
- Bearer token authentication via headers
- Connection reuse (session-based, no new connection per request)

### Non-Functional

- <50ms p99 latency for single operations
- Graceful degradation on network failure
- Thread-safe client operations
- Clear error messages on failures

## Implementation Approach

1. Create `src/adapters/redis_adapter.py`:
   - `RedisClient` class with:
     - `__init__(url, token)`: initialize with Upstash credentials
     - `get(key)` → value or None
     - `set(key, value, ex=None)` → bool
     - `delete(key)` → bool
     - `rpush(key, value)` → int (new length)
     - `lpush(key, value)` → int (new length)
     - `lrange(key, start, end)` → list
     - `ltrim(key, start, end)` → bool
     - `ttl(key)` → int (seconds, -1 if no expire, -2 if not exist)
     - `mget(keys)` → dict
     - `exists(key)` → bool
   - `@retry_with_backoff` decorator for automatic retry on transient failures
   - Request validation and error logging

2. Update `src/config.py`:
   - `UPSTASH_REDIS_REST_URL = os.getenv("UPSTASH_REDIS_REST_URL")`
   - `UPSTASH_REDIS_REST_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN")`
   - `Config.validate()` checks both required at startup

3. Implementation notes:
   - Use `requests.Session()` for connection pooling
   - All operations type-hinted
   - No dependencies on redis-py
   - All operations return parsed JSON responses

## Acceptance Criteria

- [ ] RedisClient implements all 10 operations
- [ ] Retry logic correctly implements exponential backoff (100ms, 200ms, 400ms)
- [ ] Connection pooling verified (same session reused across calls)
- [ ] Timeout handling graceful (raises RedisTimeoutError)
- [ ] All operations <50ms p99 with real Upstash instance
- [ ] Error responses properly converted to exceptions
- [ ] JSON serialization/deserialization for all operations
- [ ] Bearer token correctly in Authorization header
- [ ] Config validates required env vars at startup
- [ ] All tests passing (15+ tests)
- [ ] lsp_diagnostics clean

## Related Requirements

- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant, workflow, and thread isolation must apply to Redis state
- FR-DATA-001 [MVP]: Real Redis integration for all state storage

## Deliverables

1. `src/adapters/redis_adapter.py` (150 LOC) - RedisClient class
2. `src/config.py` (update with env var loading and validation)
3. `tests/test_redis_adapter.py` (200+ LOC, 15+ tests)

## Test Coverage (15+ tests)

### Unit Tests (no Redis needed)

- [ ] Config loads UPSTASH_REDIS_REST_URL from environment
- [ ] Config loads UPSTASH_REDIS_REST_TOKEN from environment
- [ ] Config.validate() raises if UPSTASH_REDIS_REST_URL missing
- [ ] Config.validate() raises if UPSTASH_REDIS_REST_TOKEN missing
- [ ] RedisClient formats URL correctly for POST
- [ ] RedisClient sets Authorization header correctly
- [ ] RedisClient formats JSON payload correctly for SET
- [ ] RedisClient parses JSON response correctly for GET

### Integration Tests (with real Upstash instance)

- [ ] SET/GET roundtrip with string value
- [ ] SET/GET roundtrip with JSON value
- [ ] DELETE removes key successfully
- [ ] GET on deleted key returns None
- [ ] EXISTS returns True/False correctly
- [ ] TTL returns correct remaining seconds
- [ ] RPUSH appends to list
- [ ] LRANGE retrieves all list items in order
- [ ] LTRIM truncates list correctly
- [ ] MGET retrieves multiple keys
- [ ] Retry logic: simulated timeout triggers retry and succeeds
- [ ] Retry logic: 3 failures raises exception
- [ ] Error handling: 5xx response raises RedisError
- [ ] Error handling: Connection refused graceful
- [ ] Concurrent requests use same connection (verify via logging)

## Error Scenarios

- [ ] Network timeout (500ms) → retry with backoff
- [ ] 500 error → retry
- [ ] 503 error → retry
- [ ] Malformed JSON response → clear exception
- [ ] Empty key → raises ValueError
- [ ] Non-JSON serializable value → raises TypeError

## Environment Variables

- `UPSTASH_REDIS_REST_URL` (required)
- `UPSTASH_REDIS_REST_TOKEN` (required)

**Reference**: See `[[../../../../../../README.md#environment-configuration-rules]]` for configuration strategy.

## Implementation Notes

- Use `requests.Session()` for connection pooling
- Implement `@retry_with_backoff` decorator with configurable backoff
- All operations should be type-hinted (use TypedDict or dataclasses)
- No dependencies on redis-py
- Log all retries for debugging (but don't spam)
- This is foundational: all other tasks depend on success here

## Testing Strategy

- Unit: Key naming and request formatting without network calls
- Integration: Real Upstash instance (use .env.local credentials)
- Concurrency: Multiple threads accessing same client
- Error scenarios: Simulated timeouts, invalid responses, network failures
