# VERIFY-GW-SKILL-101-FUNC: Invalidation API shape

> **Linked Task** : GW-SKILL-101 — `docs/graph-weave/specification/skills/tasks/MOCK/GW-SKILL-101.md`
> **Verification Type** : FUNC
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 2. Scope Compliance

- The mock app must expose an invalidation API with tenant, skill, and reason.

## 3. Type-Specific Criteria

| #       | Criterion          | Expected                                   | Actual | Status      |
| ------- | ------------------ | ------------------------------------------ | ------ | ----------- |
| FUNC-01 | API inputs present | Tenant, skill identifier, and reason exist |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 5. Final Decision

| Decision        | Condition                        |
| --------------- | -------------------------------- |
| Pass            | Invalidation inputs are explicit |
| Needs Revision  | API inputs are incomplete        |
| Fail + Rollback | API conflicts with skill spec    |

**Decision:** Pending
