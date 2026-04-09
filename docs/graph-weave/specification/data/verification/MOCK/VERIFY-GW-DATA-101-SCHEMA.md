# VERIFY-GW-DATA-101-SCHEMA: Versioned key shape

> **Linked Task** : GW-DATA-101 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-101.md`
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

- The cache key must include tenant, skill, and version.

## 3. Type-Specific Criteria

| #         | Criterion           | Expected                                      | Actual | Status      |
| --------- | ------------------- | --------------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Versioned key shape | Cache key includes tenant, skill, and version |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                       |
| --------------- | ------------------------------- |
| Pass            | Versioned key shape is explicit |
| Needs Revision  | Key shape is ambiguous          |
| Fail + Rollback | Key model conflicts with spec   |

**Decision:** Pending
