# VERIFY-GW-ARCH-102-FUNC: Dynamic OpenAPI output

> **Linked Task** : GW-ARCH-102 — `docs/graph-weave/specification/architecture/tasks/MOCK/GW-ARCH-102.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/architecture/system-architecture.md`
- `docs/graph-weave/specification/architecture/macro-architecture.md`

## 2. Scope Compliance

- The mock app must render params, body, and descriptions in generated OpenAPI docs.

## 3. Type-Specific Criteria

| #       | Criterion              | Expected                                      | Actual | Status      |
| ------- | ---------------------- | --------------------------------------------- | ------ | ----------- |
| FUNC-01 | OpenAPI fields present | Params, body, and descriptions appear in docs |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/architecture/system-architecture.md`

## 5. Final Decision

| Decision        | Condition                               |
| --------------- | --------------------------------------- |
| Pass            | Generated docs show the required fields |
| Needs Revision  | One required field is missing           |
| Fail + Rollback | Output conflicts with the spec          |

**Decision:** Pending
