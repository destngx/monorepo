# GW-DATA-103: Add mock Redis adapter

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Redis lookup cache
- **Feature** : Mock Redis adapter
- **Rationale** : MOCK needs one fake storage layer so cache-dependent behavior can run without a live Redis dependency

---

### Input

- **Data / Files** : `[[../../data/redis-namespace-design.md]]`, `[[../../data/compiled-graph-and-checkpoint-storage.md]]`
- **Dependencies** : GW-ARCH-103
- **External Systems**: Redis-compatible in-memory store

---

### Scope

- **In Scope** :
  - Create a mock Redis adapter for cache operations
  - Keep the adapter compatible with the documented namespace rules

- **Out of Scope**:
  - Production Redis tuning
  - Tenant sharding
  - Real database connectivity

- **Max Increment**: One runnable mock Redis adapter

---

### Approach

1. Build the fake Redis adapter
2. Keep the adapter aligned with the namespace documentation

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/redis-namespace-design.md` — source of truth for key families
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-103-FUNC.md` — verify cache operations work through the mock adapter
- `docs/graph-weave/specification/data/verification/MOCK/VERIFY-GW-DATA-103-DOC.md` — verify the adapter contract stays documented

---

### Expected Output

- **Deliverable** : Working mock Redis adapter
- **Format** : Adapter code + documentation update
- **Example** : in-memory get/set/delete support

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-DATA-103-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-DATA-103-DOC.md]]

---

### References

[[../../data/redis-namespace-design.md]] - Defines the Redis key families and cache contract that the mock adapter must honor.
