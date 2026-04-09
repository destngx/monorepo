# GW-RUNTIME-105: Add mock checkpoint storage

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime control flow
- **Feature** : Mock checkpoint storage
- **Rationale** : MOCK needs one in-memory state layer so reruns and resumes can behave like a stateful workflow

---

### Input

- **Data / Files** : `[[../../data/compiled-graph-and-checkpoint-storage.md]]`, `[[../../runtime/request-lifecycle.md]]`
- **Dependencies** : GW-DATA-103
- **External Systems**: In-memory checkpoint store

---

### Scope

- **In Scope** :
  - Add in-memory checkpoint storage for mock executions
  - Keep the state reusable across a rerun or resume path

- **Out of Scope**:
  - Durable persistence
  - Schema migrations
  - Multi-node replication

- **Max Increment**: One runnable mock checkpoint path

---

### Approach

1. Add the checkpoint storage adapter
2. Keep the checkpoint lifecycle aligned with the runtime docs

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/compiled-graph-and-checkpoint-storage.md` — source of truth for checkpoint behavior
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-105-FUNC.md` — verify checkpoint storage works
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-105-DOC.md` — verify the checkpoint rule stays documented

---

### Expected Output

- **Deliverable** : Working mock checkpoint storage
- **Format** : Adapter code + documentation update
- **Example** : checkpoint can be stored and read back in memory

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-RUNTIME-105-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-RUNTIME-105-DOC.md]]

---

### References

[[../../data/compiled-graph-and-checkpoint-storage.md]] - Defines the storage contract that the mock checkpoint path must reflect.
