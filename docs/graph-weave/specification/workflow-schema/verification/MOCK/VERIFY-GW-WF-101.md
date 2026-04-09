# VERIFY-GW-WF-101-SCHEMA-DOC: Mock workflow-doc labels

> **Linked Task** : GW-WF-101 — `docs/graph-weave/specification/workflow-schema/tasks/MOCK/GW-WF-101.md`
> **Verification Types** : SCHEMA, DOC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md`
- `docs/graph-weave/specification/README.md`

## 2. Scope Compliance

- The mock workflow docs must support inline phase labels on requirements.
- Multiple labels must be allowed for one requirement.

## 3. Type-Specific Criteria

### SCHEMA

| #         | Criterion           | Expected                                             | Actual | Status      |
| --------- | ------------------- | ---------------------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Label format        | Requirements can show `[MOCK,MVP,FULL]` style tags   |        | in progress |
| SCHEMA-02 | Multi-label support | A single requirement can carry multiple phase labels |        | in progress |

### DOC

| #      | Criterion              | Expected                                               | Actual | Status      |
| ------ | ---------------------- | ------------------------------------------------------ | ------ | ----------- |
| DOC-01 | Phase intent preserved | The workflow docs still explain the mock phase clearly |        | in progress |
| DOC-02 | Label rule traceable   | The label convention is anchored in the spec README    |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/README.md`

## 5. Final Decision

| Decision        | Condition                                         |
| --------------- | ------------------------------------------------- |
| Pass            | Label convention is explicit and reusable         |
| Needs Revision  | The label format is ambiguous                     |
| Fail + Rollback | Labels conflict with the workflow schema contract |

**Decision:** Pending
