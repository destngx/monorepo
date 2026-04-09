# VERIFY-GW-WF-101-SCHEMA: Single-label workflow requirement

> **Linked Task** : GW-WF-101 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-101.md`
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

- The mock workflow docs must allow one phase label on a requirement.

## 3. Type-Specific Criteria

| #         | Criterion           | Expected                            | Actual | Status      |
| --------- | ------------------- | ----------------------------------- | ------ | ----------- |
| SCHEMA-01 | Single-label format | Requirements can show `[MOCK]` tags |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/README.md`

## 5. Final Decision

| Decision        | Condition                          |
| --------------- | ---------------------------------- |
| Pass            | Single-label format is explicit    |
| Needs Revision  | Label format is ambiguous          |
| Fail + Rollback | Label format conflicts with schema |

**Decision:** Pending
