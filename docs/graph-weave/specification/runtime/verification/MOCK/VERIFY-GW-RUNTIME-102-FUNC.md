# VERIFY-GW-RUNTIME-102-FUNC: New thread_id on rerun

> **Linked Task** : GW-RUNTIME-102 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-102.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-09
> **Overall Status** : Pass

---

## 1. Traceability

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 2. Scope Compliance

- The mock runtime must emit a new `thread_id` on rerun.

## 3. Type-Specific Criteria

| #       | Criterion            | Expected                              | Actual | Status   |
| ------- | -------------------- | ------------------------------------- | ------ | -------- |
| FUNC-01 | New attempt identity | Reruns return a different `thread_id` | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | `thread_id` changes on rerun          |
| Needs Revision  | Attempt identity is not separated     |
| Fail + Rollback | Behavior conflicts with lifecycle doc |

**Decision:** Pass

## 6. Evidence

- **Test File**: `tests/test_runtime_stable_id.py`
- **Key Test**: `test_rerun_preserves_run_id` (line 50) verifies `thread_id_second != thread_id_first`
- **Implementation**: `apps/graph-weave/src/main.py` line 46 - `thread_id = str(uuid.uuid4())` always generates fresh ID
- **Result**: 3/3 tests passing, including thread_id uniqueness on rerun
- **All Tests**: 49/49 passing
