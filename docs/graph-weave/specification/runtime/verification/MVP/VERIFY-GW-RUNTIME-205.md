# VERIFY-GW-RUNTIME-205-FUNC: Workflow Create Hardening

> **Linked Task** : GW-MVP-RUNTIME-205 — `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-205.md`
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

- The docs must keep workflow creation validation explicit.
- The docs must keep persistence and audit requirements visible.

## 3. Type-Specific Criteria

| #       | Criterion         | Expected                                       | Actual | Status      |
| ------- | ----------------- | ---------------------------------------------- | ------ | ----------- |
| FUNC-01 | Authz             | Create requires authn/authz                    |        | in progress |
| FUNC-02 | Tenant ownership  | Tenant ownership is verified                   |        | in progress |
| FUNC-03 | Schema validation | Workflow definition is validated               |        | in progress |
| FUNC-04 | Uniqueness        | Duplicate workflow IDs/names are rejected      |        | in progress |
| FUNC-05 | Persistence       | PostgreSQL persistence and audit logging noted |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/runtime/tasks/MVP/GW-MVP-RUNTIME-205.md`

## 5. Final Decision

| Decision        | Condition                               |
| --------------- | --------------------------------------- |
| Pass            | Create contract is explicit             |
| Needs Revision  | Tenant or uniqueness wording is unclear |
| Fail + Rollback | Create contract conflicts with spec     |

**Decision:** Pending
