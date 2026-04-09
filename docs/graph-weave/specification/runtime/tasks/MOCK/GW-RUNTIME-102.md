# GW-RUNTIME-102: Add new thread_id on rerun

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime control flow
- **Feature** : New internal attempt identity in the mock app
- **Rationale** : MOCK needs one separate attempt identifier so reruns can be tracked internally

---

### Input

- **Data / Files** : `[[../../runtime/request-lifecycle.md]]`, `[[../../runtime/plan/request-lifecycle-and-streaming.md]]`, `[[../../runtime/README.md]]`
- **Dependencies** : GW-RUNTIME-101
- **External Systems**: FastAPI gateway

---

### Scope

- **In Scope** :
  - Return a new `thread_id` on rerun
  - Keep the attempt identity visible in the mock API response

- **Out of Scope**:
  - Stable `run_id` behavior
  - Production retry orchestration
  - UI behavior for retry controls

- **Max Increment**: One working mock attempt identity response

---

### Approach

1. Add a mock rerun path that emits a new `thread_id`
2. Keep the mock flow aligned with the request lifecycle docs

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/request-lifecycle.md` — source of truth for attempt identity behavior
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-102-FUNC.md` — verify `thread_id` changes on rerun
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-102-DOC.md` — verify the attempt-identity rule stays documented

---

### Expected Output

- **Deliverable** : Working mock attempt identity response
- **Format** : App code + documentation update
- **Example** : new `thread_id` returned on rerun

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-RUNTIME-102-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-RUNTIME-102-DOC.md]]

---

### References

[[../../runtime/request-lifecycle.md]] - Contains the run/thread split and the retry history note that this task must preserve.
