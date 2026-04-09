# GW-SKILL-101: Add cache invalidation API

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Skill discovery and loading
- **Feature** : External skill cache invalidation API for the mock app
- **Rationale** : MOCK needs one API entrypoint for external edits before cache miss rebuild behavior is added

---

### Input

- **Data / Files** : `[[../../skills/llm-skills-architecture.md]]`, `[[../../skills/skill-loading-flow.md]]`, `[[../../skills/plan/skill-registry-and-metadata-contracts.md]]`
- **Dependencies** : GW-SKILL-001
- **External Systems**: Redis, external tooling/operators

---

### Scope

- **In Scope** :
  - Implement invalidation trigger inputs: tenant scope, skill identifier, reason
  - Expose the invalidation flow as part of the working mock app

- **Out of Scope**:
  - Cache miss rebuild behavior
  - Production auth/authorization hardening
  - Tooling UI polish

- **Max Increment**: One working mock invalidation endpoint

---

### Approach

1. Add an invalidation API for tenant-scoped skill cache entries
2. Keep the API aligned with the skill-loading docs

**Files to Modify/Create**:

- `docs/graph-weave/specification/skills/llm-skills-architecture.md` — source of truth for the invalidation API contract
- `docs/graph-weave/specification/skills/skill-loading-flow.md` — source of truth for cache miss and reload behavior
- `docs/graph-weave/specification/skills/verification/MOCK/VERIFY-GW-SKILL-101-FUNC.md` — verify the invalidation API shape
- `docs/graph-weave/specification/skills/verification/MOCK/VERIFY-GW-SKILL-101-DOC.md` — verify the invalidation rule stays documented

---

### Expected Output

- **Deliverable** : Working mock invalidation API
- **Format** : App code + documentation update
- **Example** : `invalidate(tenant_id, skill_id, reason)`

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-SKILL-101-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-SKILL-101-DOC.md]]

---

### References

[[../../skills/llm-skills-architecture.md]] - Contains the cache note and invalidation expectations for external edits.
