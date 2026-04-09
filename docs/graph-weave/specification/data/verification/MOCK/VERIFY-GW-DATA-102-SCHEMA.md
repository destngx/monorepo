# VERIFY-GW-DATA-102-SCHEMA: Latest fallback rule

> **Linked Task** : GW-DATA-102 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-102.md`
> **Verification Type** : SCHEMA
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 2. Scope Compliance

- Missing version lookups must resolve to `latest`.

## 3. Type-Specific Criteria

| #         | Criterion      | Expected                             | Actual | Status      |
| --------- | -------------- | ------------------------------------ | ------ | ----------- |
| SCHEMA-01 | Latest default | Missing version resolves to `latest` |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                          |
| --------------- | ---------------------------------- |
| Pass            | Latest fallback is explicit        |
| Needs Revision  | Fallback rule is ambiguous         |
| Fail + Rollback | Fallback conflicts with cache spec |

**Decision:** Pending
