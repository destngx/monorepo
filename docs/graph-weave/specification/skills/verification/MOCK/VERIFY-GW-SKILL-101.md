# VERIFY-GW-SKILL-101-FUNC-SCHEMA: Skill cache invalidation API

> **Linked Task** : GW-SKILL-101 — `docs/graph-weave/specification/skills/tasks/MOCK/GW-SKILL-101.md`
> **Verification Types** : FUNC, SCHEMA
> **Phase ID** : MOCK
> **Risk Level** : High
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`
- `docs/graph-weave/specification/skills/skill-loading-flow.md`
- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 2. Scope Compliance

- The docs must define an invalidation API.
- The docs must specify tenant scope, skill identifier, and reason as inputs.

## 3. Type-Specific Criteria

### FUNC

| #       | Criterion                  | Expected                                                    | Actual | Status      |
| ------- | -------------------------- | ----------------------------------------------------------- | ------ | ----------- |
| FUNC-01 | Invalidation trigger shape | API accepts tenant scope, skill identifier, reason          |        | in progress |
| FUNC-02 | Cache-miss behavior        | Cache miss rebuilds from folder/frontmatter source of truth |        | in progress |

### SCHEMA

| #         | Criterion             | Expected                                             | Actual | Status      |
| --------- | --------------------- | ---------------------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Versioned cache keys  | Lookup keys include version with `latest` default    |        | in progress |
| SCHEMA-02 | External edit support | Invalidation is represented as an API, not a restart |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/skills/llm-skills-architecture.md`

## 5. Final Decision

| Decision        | Condition                                       |
| --------------- | ----------------------------------------------- |
| Pass            | API inputs and cache miss behavior are explicit |
| Needs Revision  | API or recovery behavior is unclear             |
| Fail + Rollback | API conflicts with skill loading contract       |

**Decision:** Pending
