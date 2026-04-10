# VERIFY-GW-RUNTIME-208-FUNC: Workflow Delete Semantics

> **Linked Task** : GW-MVP-RUNTIME-208 — `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-208.md`
> **Verification Type** : FUNC
> **Phase ID** : MVP
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-10 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_MANAGEMENT_API.md`
- `docs/graph-weave/src/main.py`

## 2. Scope Compliance

- The docs must keep delete authorization explicit.
- The docs must keep dependency, cleanup, and soft-delete behavior visible.

## 3. Type-Specific Criteria

| #       | Criterion         | Expected                                 | Actual | Status      |
| ------- | ----------------- | ---------------------------------------- | ------ | ----------- |
| FUNC-01 | Authorization     | Delete requests require authorization    |        | in progress |
| FUNC-02 | Dependency checks | Referenced workflows are protected       |        | in progress |
| FUNC-03 | Active executions | Active executions block or gate deletion |        | in progress |
| FUNC-04 | Soft delete       | Soft-delete semantics are documented     |        | in progress |
| FUNC-05 | Audit logging     | Delete actions are auditable             |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-208.md`

## 5. Final Decision

| Decision        | Condition                                 |
| --------------- | ----------------------------------------- |
| Pass            | Delete semantics are explicit             |
| Needs Revision  | Soft-delete or dependency wording unclear |
| Fail + Rollback | Delete contract conflicts with spec       |

**Decision:** Pending
