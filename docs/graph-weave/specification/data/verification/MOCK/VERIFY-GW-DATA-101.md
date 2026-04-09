# VERIFY-GW-DATA-101-SCHEMA-FUNC: Versioned skill cache keys

> **Linked Task** : GW-DATA-101 — `docs/graph-weave/specification/data/tasks/MOCK/GW-DATA-101.md`
> **Verification Types** : SCHEMA, FUNC
> **Phase ID** : MOCK
> **Risk Level** : Medium
> **Reviewer** : Hephaestus
> **Verified On** : 2026-04-08 00:00
> **Overall Status** : Pending

---

## 1. Traceability

- `docs/graph-weave/specification/data/redis-namespace-design.md`
- `docs/graph-weave/specification/skills/llm-skills-architecture.md`
- `docs/graph-weave/specification/skills/skill-loading-flow.md`

## 2. Scope Compliance

- Skill cache keys must include a version segment.
- `latest` must be the default lookup target when version is not specified.

## 3. Type-Specific Criteria

### SCHEMA

| #         | Criterion           | Expected                                      | Actual | Status      |
| --------- | ------------------- | --------------------------------------------- | ------ | ----------- |
| SCHEMA-01 | Versioned key shape | Cache key includes tenant, skill, and version |        | in progress |
| SCHEMA-02 | Latest default      | Missing version resolves to `latest`          |        | in progress |

### FUNC

| #       | Criterion                  | Expected                                                  | Actual | Status      |
| ------- | -------------------------- | --------------------------------------------------------- | ------ | ----------- |
| FUNC-01 | Mock lookup path present   | The mock app exposes a version-aware lookup flow          |        | in progress |
| FUNC-02 | Invalidation compatibility | Version changes remain compatible with invalidation rules |        | in progress |

## 4. Documentation Check

- `docs/graph-weave/specification/data/redis-namespace-design.md`

## 5. Final Decision

| Decision        | Condition                               |
| --------------- | --------------------------------------- |
| Pass            | Key format and default are explicit     |
| Needs Revision  | Version or default is ambiguous         |
| Fail + Rollback | Key model conflicts with cache contract |

**Decision:** Pending
