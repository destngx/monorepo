# VERIFY-GW-RUNTIME-101-FUNC: Stable run_id on rerun

> **Linked Task** : GW-RUNTIME-101 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-101.md`
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

- The mock runtime must keep `run_id` stable across reruns.

## 3. Type-Specific Criteria

| #       | Criterion       | Expected                        | Actual | Status   |
| ------- | --------------- | ------------------------------- | ------ | -------- |
| FUNC-01 | Stable identity | Reruns return the same `run_id` | Pass   | complete |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | `run_id` stays stable                 |
| Needs Revision  | Identity is still ambiguous           |
| Fail + Rollback | Behavior conflicts with lifecycle doc |

**Decision:** Pass

## 6. Evidence

- **Test File**: `tests/test_runtime_stable_id.py`
- **Tests**: 3/3 passing
  - `test_rerun_preserves_run_id`: Verifies run_id stays stable when rerun with same run_id, new thread_id generated
  - `test_new_run_generates_new_run_id`: Verifies normal executions still generate unique run_ids
  - `test_rerun_response_structure`: Verifies response structure includes all required fields on rerun
- **Implementation**: `apps/graph-weave/src/main.py` line 46-47 - ExecuteRequest accepts optional run_id, execute() preserves or generates
- **All Tests**: 49/49 passing (46 existing + 3 new)
