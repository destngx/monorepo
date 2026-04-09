# VERIFY-GW-DATA-101-FUNC: Versioned key used by mock app

> **Linked Task** : GW-DATA-101 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-101.md`
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

- The mock app must use the versioned key shape.

## 3. Type-Specific Criteria

| #       | Criterion      | Expected                                | Actual | Status      |
| ------- | -------------- | --------------------------------------- | ------ | ----------- |
| FUNC-01 | Key shape used | Mock app reads/writes the versioned key |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                             |
| --------------- | ------------------------------------- |
| Pass            | Mock app uses the versioned key shape |
| Needs Revision  | Lookup still uses the wrong key shape |
| Fail + Rollback | Behavior conflicts with cache spec    |

**Decision:** Pending
