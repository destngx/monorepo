# GW-DATA-102: Add latest skill fallback

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Redis lookup cache
- **Feature** : Latest fallback for skill cache lookup in the mock app
- **Rationale** : MOCK needs one default lookup rule so versionless requests still resolve predictably

---

### Input

- **Data / Files** : `[[../../data/redis-namespace-design.md]]`, `[[../../skills/llm-skills-architecture.md]]`, `[[../../skills/skill-loading-flow.md]]`
- **Dependencies** : GW-DATA-101
- **External Systems**: Redis

---

### Scope

- **In Scope** :
  - Default missing version lookups to `latest`
  - Keep the lookup behavior compatible with the mock cache key shape

- **Out of Scope**:
  - Versioned key shape
  - Production cache tuning
  - Skill authoring UX

- **Max Increment**: One working mock latest fallback rule

---

### Approach

1. Add the `latest` default lookup rule
2. Keep the fallback behavior aligned with the Redis namespace doc

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/redis-namespace-design.md` — source of truth for the default lookup rule
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-102-SCHEMA.md` — verify the fallback key rule
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-102-FUNC.md` — verify the mock app uses the fallback rule

---

### Expected Output

- **Deliverable** : Working mock latest fallback rule
- **Format** : App code + documentation update
- **Example** : missing version resolves to `latest`

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-DATA-102-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-DATA-102-FUNC.md]]

---

### References

[[../../data/redis-namespace-design.md]] - Contains the key family and versioning rules that define the fallback contract.
