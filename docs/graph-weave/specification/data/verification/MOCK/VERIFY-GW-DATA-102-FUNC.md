# VERIFY-GW-DATA-102-FUNC: Latest fallback used by mock app

> **Linked Task** : GW-DATA-102 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-102.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 2. Scope Compliance

- The mock app must use `latest` when version is omitted.

## 3. Type-Specific Criteria

| #       | Criterion       | Expected                                 | Actual | Status      |
| ------- | --------------- | ---------------------------------------- | ------ | ----------- |
| FUNC-01 | Fallback active | Versionless lookup resolves via `latest` |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                          |
| --------------- | ---------------------------------- |
| Pass            | Mock app uses the latest fallback  |
| Needs Revision  | Fallback is not applied            |
| Fail + Rollback | Behavior conflicts with cache spec |

**Decision:** Pending
