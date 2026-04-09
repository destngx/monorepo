# VERIFY-GW-WF-103-SCHEMA: Production workflow contract

> **Linked Task** : GW-WF-103 — `docs/graph-weave/specification/workflow-schema/tasks/FULL/GW-WF-103.md`
> **Verification Type** : SCHEMA
> **Phase ID** : FULL
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md`
- `docs/graph-weave/specification/workflow-schema/plan/schema-and-node-contracts.md`

## 2. Scope Compliance

- The docs must keep node, edge, and guardrail contracts explicit.
- The docs must keep versioning and migration rules visible.

## 3. Type-Specific Criteria

| #         | Criterion            | Expected                                      | Actual | Status      |
| --------- | -------------------- | --------------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Node/edge contract   | Production workflow contract remains explicit |        | in progress |
| SCHEMA-02 | Guardrail/versioning | Guardrails and versioning stay visible        |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md`

## 5. Final Decision

| Decision        | Condition                                   |
| --------------- | ------------------------------------------- |
| Pass            | Production workflow contract is explicit    |
| Needs Revision  | Contract wording is ambiguous               |
| Fail + Rollback | Contract conflicts with the workflow schema |

**Decision:** Pending
