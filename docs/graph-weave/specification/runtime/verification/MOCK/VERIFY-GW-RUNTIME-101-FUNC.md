# VERIFY-GW-RUNTIME-101-FUNC: Stable run_id on rerun

> **Linked Task** : GW-RUNTIME-101 — `docs/graph-weave/specification/runtime/tasks/MOCK/GW-RUNTIME-101.md`
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

- The mock runtime must keep `run_id` stable across reruns.

## 3. Type-Specific Criteria

| #       | Criterion       | Expected                        | Actual | Status      |
| ------- | --------------- | ------------------------------- | ------ | ----------- |
| FUNC-01 | Stable identity | Reruns return the same `run_id` |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/request-lifecycle.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | `run_id` stays stable                 |
| Needs Revision  | Identity is still ambiguous           |
| Fail + Rollback | Behavior conflicts with lifecycle doc |

**Decision:** Pending
