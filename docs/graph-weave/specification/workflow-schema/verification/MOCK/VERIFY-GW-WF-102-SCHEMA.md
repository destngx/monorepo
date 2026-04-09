# VERIFY-GW-WF-102-SCHEMA: Multi-label workflow requirement

> **Linked Task** : GW-WF-102 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-102.md`
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md`

## 2. Scope Compliance

- The mock workflow docs must allow multiple labels on one requirement.

## 3. Type-Specific Criteria

| #         | Criterion          | Expected                                | Actual | Status      |
| --------- | ------------------ | --------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Multi-label format | Requirements can show `[MOCK,MVP,FULL]` |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/README.md`

## 5. Final Decision

| Decision        | Condition                          |
| --------------- | ---------------------------------- |
| Pass            | Multi-label format is explicit     |
| Needs Revision  | Label combination is ambiguous     |
| Fail + Rollback | Label format conflicts with schema |

**Decision:** Pending
