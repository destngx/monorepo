# VERIFY-GW-RUNTIME-207-FUNC: Workflow Update Semantics

> **Linked Task** : GW-MVP-RUNTIME-207 — `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-207.md`
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

- The docs must keep immutable-field and versioning behavior explicit.
- The docs must keep recompilation and audit expectations visible.

## 3. Type-Specific Criteria

| #       | Criterion         | Expected                                 | Actual | Status      |
| ------- | ----------------- | ---------------------------------------- | ------ | ----------- |
| FUNC-01 | Immutable fields  | Immutable fields cannot change           |        | in progress |
| FUNC-02 | Authorization     | Update requests require proper authz     |        | in progress |
| FUNC-03 | Version history   | Update creates version history entry     |        | in progress |
| FUNC-04 | Recompile trigger | Definition changes trigger recompilation |        | in progress |
| FUNC-05 | Audit logging     | Update actions are auditable             |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-207.md`

## 5. Final Decision

| Decision        | Condition                              |
| --------------- | -------------------------------------- |
| Pass            | Update semantics are explicit          |
| Needs Revision  | Versioning or recompilation is unclear |
| Fail + Rollback | Update contract conflicts with spec    |

**Decision:** Pending
