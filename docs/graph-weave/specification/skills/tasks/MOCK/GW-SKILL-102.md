# GW-SKILL-102: Rebuild cache on miss

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Skill discovery and loading
- **Feature** : Cache miss rebuild for skills in the mock app
- **Rationale** : MOCK needs one recovery path so a missing cache entry reloads from the folder/frontmatter source of truth

---

### Input

- **Data / Files** : `[[../../skills/llm-skills-architecture.md]]`, `[[../../skills/skill-loading-flow.md]]`, `[[../../skills/plan/skill-registry-and-metadata-contracts.md]]`
- **Dependencies** : GW-SKILL-101
- **External Systems**: Redis, external tooling/operators

---

### Scope

- **In Scope** :
  - Rebuild the cache from folder/frontmatter source of truth on miss
  - Expose the cache-miss recovery flow as part of the working mock app

- **Out of Scope**:
  - Invalidation API shape
  - Production auth/authorization hardening
  - Tooling UI polish

- **Max Increment**: One working mock cache miss recovery path

---

### Approach

1. Rebuild the skill entry when the cache misses
2. Keep the reload flow aligned with the skill-loading docs

**Files to Modify/Create**:

- `docs/graph-weave/specification/skills/skill-loading-flow.md` — source of truth for miss recovery behavior
- `docs/graph-weave/specification/skills/verification/MOCK/VERIFY-GW-SKILL-102-FUNC.md` — verify the cache miss rebuild flow
- `docs/graph-weave/specification/skills/verification/MOCK/VERIFY-GW-SKILL-102-DOC.md` — verify the rebuild rule stays documented

---

### Expected Output

- **Deliverable** : Working mock cache miss rebuild
- **Format** : App code + documentation update
- **Example** : cache miss loads the skill from frontmatter source

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-SKILL-102-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-SKILL-102-DOC.md]]

---

### References

[[../../skills/skill-loading-flow.md]] - Contains the cache miss and reload behavior that this task must preserve.
