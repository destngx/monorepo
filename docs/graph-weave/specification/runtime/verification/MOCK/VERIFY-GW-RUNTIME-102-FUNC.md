# VERIFY-GW-RUNTIME-102-FUNC: New thread_id on rerun

> **Linked Task** : GW-RUNTIME-102 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-102.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 2. Scope Compliance

- The mock runtime must emit a new `thread_id` on rerun.

## 3. Type-Specific Criteria

| #       | Criterion            | Expected                              | Actual | Status      |
| ------- | -------------------- | ------------------------------------- | ------ | ----------- |
| FUNC-01 | New attempt identity | Reruns return a different `thread_id` |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | `thread_id` changes on rerun          |
| Needs Revision  | Attempt identity is not separated     |
| Fail + Rollback | Behavior conflicts with lifecycle doc |

**Decision:** Pending
