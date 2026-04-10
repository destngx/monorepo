# MVP Task Index: Complete 14-Task Specification

**Purpose**: Central registry of all 14 MVP tasks with explicit dependencies, timing, and ordering for parallel execution.

**Generated**: From dependency analysis and task splitting recommendations (DATA-201, RUNTIME-202 split from 11 → 14 tasks)

**Critical Path**: 8 hours minimum (DATA-201A → DATA-201B → RUNTIME-202B → RUNTIME-202C → E2E-002)

---

## All 14 MVP Tasks (Execution Order)

### Foundation Phase (2-3h, SEQUENTIAL)

#### GW-MVP-DATA-201A: Redis Connection & Adapter (1.5h)

- **Objective**: HTTP-based Redis client via Upstash REST API
- **Blocked By**: NONE (foundational)
- **Blocks**: DATA-201B, RUNTIME-201, RUNTIME-202A/B/C, DATA-202, DATA-203, E2E-001/002/003
- **Deliverables**: `src/adapters/redis_adapter.py`, `tests/test_redis_adapter.py` (15+ tests)
- **File**: `[[data/tasks/MVP/GW-MVP-DATA-201A.md]]`

#### GW-MVP-DATA-201B: Redis Namespacing & Circuit Breaker (1.5h)

- **Objective**: Tenant-scoped keys and circuit breaker pattern for reliability
- **Blocked By**: DATA-201A
- **Blocks**: RUNTIME-201, RUNTIME-202A/B/C, DATA-202, DATA-203, E2E-001/002/003
- **Deliverables**: `src/adapters/redis_circuit_breaker.py`, `tests/test_redis_circuit_breaker.py` (20+ tests)
- **File**: `[[data/tasks/MVP/GW-MVP-DATA-201B.md]]`

### Runtime Layer Phase (Parallel with Data Layer)

#### GW-MVP-RUNTIME-201: Status Lifecycle (1.5h)

- **Objective**: 7-state execution status (queued, validating, pending, running, completed, failed, cancelled)
- **Blocked By**: DATA-201B
- **Blocks**: RUNTIME-203, DATA-203, E2E-001, E2E-003
- **Deliverables**: `src/services/status_service.py`, `tests/test_status_lifecycle.py` (15+ tests)
- **File**: `[[runtime/tasks/MVP/GW-MVP-RUNTIME-201.md]]`

#### GW-MVP-RUNTIME-202A: LangGraph Graph Builder (1h)

- **Objective**: Build StateGraph from workflow JSON (entry, agent_node, branch, exit nodes)
- **Blocked By**: NONE (no external dependencies)
- **Blocks**: RUNTIME-202B, RUNTIME-202C
- **Deliverables**: `src/adapters/langgraph_graph_builder.py`, `tests/test_langgraph_graph_builder.py` (12+ tests)
- **File**: `[[runtime/tasks/MVP/GW-MVP-RUNTIME-202A.md]]`

#### GW-MVP-RUNTIME-202B: LLM & MCP Tool Integration (1.5h)

- **Objective**: Claude LLM integration with MCP tool routing (load_skill, search, verify)
- **Blocked By**: DATA-201B
- **Blocks**: RUNTIME-202C
- **Deliverables**: `src/adapters/mcp_router.py`, `tests/test_mcp_router.py` (15+ tests)
- **File**: `[[runtime/tasks/MVP/GW-MVP-RUNTIME-202B.md]]`

#### GW-MVP-RUNTIME-202C: Executor & Stagnation Detection (1h)

- **Objective**: Real LangGraph execution loop with event emission and stagnation detection
- **Blocked By**: RUNTIME-202A, RUNTIME-202B
- **Blocks**: DATA-202, E2E-002, E2E-003
- **Deliverables**: `src/adapters/stagnation_detector.py`, `tests/test_stagnation_detector.py` (15+ tests)
- **File**: `[[runtime/tasks/MVP/GW-MVP-RUNTIME-202C.md]]`

#### GW-MVP-RUNTIME-203: Event Emitter (1h)

- **Objective**: Real event emission and storage in Redis event log (18 event types)
- **Blocked By**: RUNTIME-201, DATA-201B
- **Blocks**: E2E-001, E2E-002, E2E-003
- **Deliverables**: `src/services/event_emitter.py`, `tests/test_event_emitter.py` (20+ tests)
- **File**: `[[runtime/tasks/MVP/GW-MVP-RUNTIME-203.md]]`

### Data Layer Phase (Depends on Runtime Layer)

#### GW-MVP-DATA-202: Checkpoint Persistence (1.5h)

- **Objective**: Redis-backed checkpoint save/restore for workflow recovery
- **Blocked By**: DATA-201B, RUNTIME-202C
- **Blocks**: E2E-002, E2E-003
- **Deliverables**: `src/services/checkpoint_service.py`, `tests/test_checkpoint_recovery.py` (20+ tests)
- **File**: `[[data/tasks/MVP/GW-MVP-DATA-202.md]]`

#### GW-MVP-DATA-203: Active Thread Lifecycle (1.5h)

- **Objective**: Thread tracking, cancellation, and cleanup
- **Blocked By**: DATA-201B, RUNTIME-201
- **Blocks**: E2E-003
- **Deliverables**: `src/services/thread_lifecycle_service.py`, `tests/test_thread_lifecycle.py` (20+ tests)
- **File**: `[[data/tasks/MVP/GW-MVP-DATA-203.md]]`

### E2E Verification Phase (Parallel, all depend on Phase 2)

#### GW-MVP-E2E-001: Workflow Execution (Happy Path) (1.5h)

- **Objective**: Complete end-to-end workflow from submission through execution completion
- **Blocked By**: RUNTIME-201, RUNTIME-202C, RUNTIME-203
- **Blocks**: NONE
- **Deliverables**: `tests/test_e2e_workflow_execution.py` (6+ test scenarios)
- **File**: `[[verification/MVP/GW-MVP-E2E-001.md]]`

#### GW-MVP-E2E-002: Agent Execution with MCP Tools (2h)

- **Objective**: Agent calls real MCP tools (load_skill, search, verify)
- **Blocked By**: RUNTIME-202C, RUNTIME-203, DATA-202
- **Blocks**: NONE
- **Deliverables**: `tests/test_e2e_agent_execution_with_mcp.py` (6+ test scenarios)
- **File**: `[[verification/MVP/GW-MVP-E2E-002.md]]`

#### GW-MVP-E2E-003: Checkpoint Recovery & Cancellation (2h)

- **Objective**: Interrupt/recover and cancellation end-to-end
- **Blocked By**: DATA-202, DATA-203, RUNTIME-201
- **Blocks**: NONE
- **Deliverables**: `tests/test_e2e_checkpoint_recovery.py` (8+ test scenarios)
- **File**: `[[verification/MVP/GW-MVP-E2E-003.md]]`

---

## Dependency Graph

```
┌──────────────────────────────────────────────────────────────┐
│ Phase 1A: Foundation (2-3h sequential)                       │
│                                                              │
│ DATA-201A (1.5h) → DATA-201B (1.5h)                         │
│    ↓                    ↓                                    │
│    +────────────┬───────+                                   │
└────────────────┼─────────┼──────────────────────────────────┘
                 │         │
         ┌───────┴─────────┴──────────────┐
         │ Phase 2: Runtime & Data        │
         │ (Parallel 3-4h + 1.5-2h)       │
         │                                │
         ├─ RUNTIME-202A (1h)            │ (no deps)
         ├─ RUNTIME-202B (1.5h)   ← DATA-201B
         ├─ RUNTIME-201 (1.5h)    ← DATA-201B
         ├─ RUNTIME-202C (1h)     ← 202A + 202B
         ├─ RUNTIME-203 (1h)      ← 201 + DATA-201B
         ├─ DATA-202 (1.5h)       ← DATA-201B + 202C
         └─ DATA-203 (1.5h)       ← DATA-201B + 201
         │                                │
         └───────────┬────────────────────┘
                     │
         ┌───────────┴──────────────────┐
         │ Phase 3: E2E Tests           │
         │ (Parallel 5-6h)              │
         │                              │
         ├─ E2E-001 (1.5h)             │
         ├─ E2E-002 (2h)  ← DATA-202   │
         └─ E2E-003 (2h)  ← DATA-202/203
         │                              │
         └──────────────────────────────┘

Critical Path: DATA-201A (1.5h) → DATA-201B (1.5h) → RUNTIME-202B (1.5h) →
              RUNTIME-202C (1h) → E2E-002 (2h) = 8 hours minimum
```

---

## Environment Variables

**Required for all MVP tasks**:

| Variable                   | Used By                                                                       | Phase |
| -------------------------- | ----------------------------------------------------------------------------- | ----- |
| `UPSTASH_REDIS_REST_URL`   | DATA-201A/B, RUNTIME-201/202A/B/C, RUNTIME-203, DATA-202/203, E2E-001/002/003 | MVP   |
| `UPSTASH_REDIS_REST_TOKEN` | DATA-201A/B, RUNTIME-201/202A/B/C, RUNTIME-203, DATA-202/203, E2E-001/002/003 | MVP   |
| `ANTHROPIC_API_KEY`        | RUNTIME-202B, E2E-002                                                         | MVP   |
| `GITHUB_TOKEN`             | RUNTIME-202B, E2E-002                                                         | MVP   |

**Reference**: `[[../README.md#environment-configuration-rules]]`

---

## Execution Timeline

```
T+0h:     Start DATA-201A (foundation)
T+1.5h:   DATA-201A complete → Start DATA-201B + RUNTIME-202A
T+2.5h:   RUNTIME-202A complete
T+3h:     DATA-201B complete → Start RUNTIME-201 + 202B
T+4h:     RUNTIME-202B complete → Start 202C + DATA-202/203
T+5h:     RUNTIME-202C complete, DATA-202/203 progress
T+5.5h:   DATA-202 complete
T+6h:     DATA-203 complete
T+6h:     All Phase 2 complete → Start E2E-001/002/003 in parallel
T+8h:     E2E-001 complete
T+8h:     E2E-002 complete (critical path finishes)
T+8h:     E2E-003 complete
T+8h:     MVP Phase Ready ✓ (all 14 tasks + 150+ tests passing)

Wall-clock: 8 hours minimum with 3 parallel teams
```

---

## Test Coverage Summary

| Task         | Unit   | Integration | Concurrency | Error  | Total    |
| ------------ | ------ | ----------- | ----------- | ------ | -------- |
| DATA-201A    | 8      | 7           | 1           | 5      | 15+      |
| DATA-201B    | 11     | 9           | 3           | 5      | 20+      |
| RUNTIME-201  | 6      | 6           | 2           | 4      | 15+      |
| RUNTIME-202A | 12     | -           | -           | 5      | 12+      |
| RUNTIME-202B | 9      | 7           | -           | 4      | 15+      |
| RUNTIME-202C | 5      | 8           | 1           | 5      | 15+      |
| RUNTIME-203  | 5      | 8           | 4           | 4      | 20+      |
| DATA-202     | 4      | 5           | 2           | 4      | 20+      |
| DATA-203     | 3      | 7           | 3           | 5      | 20+      |
| E2E-001      | -      | 4           | 3           | 3      | 6+       |
| E2E-002      | -      | 5           | 1           | 3      | 6+       |
| E2E-003      | -      | 4           | 2           | 4      | 8+       |
| **Total**    | **63** | **71**      | **25**      | **50** | **150+** |

---

## Success Criteria

- ✅ All 14 task specifications with complete acceptance criteria
- ✅ All 14 task environment variables and configuration requirements documented
- ✅ Explicit dependencies documented for all tasks (no hidden blocking relationships)
- ✅ 150+ tests planned with comprehensive coverage (unit, integration, concurrency, error scenarios)
- ⏳ Critical path identified and minimized (8 hours with optimal parallelization)
- ⏳ All 205 MOCK phase tests still passing (no regressions)
- ⏳ 0 type errors, 0 lsp_diagnostics on new code

---

## Implementation Notes

- All 14 tasks use TDD: red → green → refactor
- Each task is independently verifiable and completable
- Concurrency tests mandatory for all tasks (no hidden race conditions)
- Error scenarios covered: timeouts, network failures, invalid data, state inconsistency
- No task has scope creep (sized 1-2h each)
- All external dependencies (Redis, Claude, MCP) required before test execution
- Redis cleanup after each test (no orphaned data)
