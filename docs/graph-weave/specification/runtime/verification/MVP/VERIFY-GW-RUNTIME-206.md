# VERIFY-GW-RUNTIME-206-FUNC: Workflow Read and List Behavior

> **Linked Task** : GW-MVP-RUNTIME-206 — `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-206.md`
> **Verification Type** : FUNC
> **Phase ID** : MVP
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-10 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_MANAGEMENT_API.md`
- `docs/graph-weave/src/main.py`

## 2. Scope Compliance

- The docs must keep read/list permission behavior explicit.
- The docs must keep pagination, sorting, and cache expectations visible.

## 3. Type-Specific Criteria

| #       | Criterion        | Expected                                 | Actual | Status      |
| ------- | ---------------- | ---------------------------------------- | ------ | ----------- |
| FUNC-01 | List permissions | Only visible workflows are returned      |        | in progress |
| FUNC-02 | Pagination       | Pagination is documented                 |        | in progress |
| FUNC-03 | Sorting          | Sorting options are documented           |        | in progress |
| FUNC-04 | Search           | Full-text search is documented           |        | in progress |
| FUNC-05 | Read extras      | Version history / stats behavior visible |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-206.md`

## 5. Final Decision

| Decision        | Condition                              |
| --------------- | -------------------------------------- |
| Pass            | Read/list behavior is explicit         |
| Needs Revision  | Permission or cache wording is unclear |
| Fail + Rollback | Read/list contract conflicts with spec |

**Decision:** Pending
