# GW-DATA-101: Add versioned cache keys

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Redis lookup cache
- **Feature** : Versioned skill cache key shape in the mock app
- **Rationale** : MOCK needs one explicit cache key shape before fallback behavior is layered on

---

### Input

- **Data / Files** : `[[../../data/redis-namespace-design.md]]`, `[[../../skills/llm-skills-architecture.md]]`, `[[../../skills/skill-loading-flow.md]]`
- **Dependencies** : GW-DATA-001, GW-SKILL-001
- **External Systems**: Redis

---

### Scope

- **In Scope** :
  - Implement versioned skill cache key behavior in the mock app
  - Include tenant, skill, and version in the cache key

- **Out of Scope**:
  - Default fallback behavior
  - Production cache tuning
  - Skill authoring UX

- **Max Increment**: One working mock cache key shape

---

### Approach

1. Implement version-aware lookup against Redis cache
2. Keep the key format compatible with external edits

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/redis-namespace-design.md` — source of truth for the cache key rules
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-101-SCHEMA.md` — verify the versioned key shape
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-101-FUNC.md` — verify the cache key shape is used by the mock app

---

### Expected Output

- **Deliverable** : Working mock versioned cache key shape
- **Format** : App code + documentation update
- **Example** : `skills:level1:{tenant_id}:{skill_id}:{version}`

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-DATA-101-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-DATA-101-FUNC.md]]

---

### References

[[../../data/redis-namespace-design.md]] - Source of truth for Redis key families and the skill cache layout.
