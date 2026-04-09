# VERIFY-GW-SKILL-103-FUNC: Production packaging contract

> **Linked Task** : GW-SKILL-103 — `docs/graph-weave/specification/skills/tasks/FULL/GW-SKILL-103.md`
> **Verification Type** : FUNC
> **Phase ID** : FULL
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`
- `docs/graph-weave/specification/skills/skill-loading-flow.md`
- `docs/graph-weave/specification/skills/plan/skill-registry-and-metadata-contracts.md`

## 2. Scope Compliance

- The docs must keep packaging, invalidation, and versioned lookup explicit.
- The docs must keep the discovery/load boundary explicit.

## 3. Type-Specific Criteria

| #       | Criterion             | Expected                                          | Actual | Status      |
| ------- | --------------------- | ------------------------------------------------- | ------ | ----------- |
| FUNC-01 | Packaging contract    | Three-level packaging remains explicit            |        | in progress |
| FUNC-02 | Invalidation contract | Versioned lookup and invalidation remain explicit |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 5. Final Decision

| Decision        | Condition                                    |
| --------------- | -------------------------------------------- |
| Pass            | Packaging contract is explicit and stable    |
| Needs Revision  | Packaging or invalidation wording is unclear |
| Fail + Rollback | Contract conflicts with skill loading        |

**Decision:** Pending
