# GW-RUNTIME-101: Preserve run_id on rerun

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime control flow
- **Feature** : Stable public run identity in the mock app
- **Rationale** : MOCK needs one stable public identifier so reruns still refer to the same run record

---

### Input

- **Data / Files** : `[[../../runtime/request-lifecycle.md]]`, `[[../../runtime/plan/request-lifecycle-and-streaming.md]]`, `[[../../runtime/README.md]]`
- **Dependencies** : GW-RUNTIME-001, GW-ARCH-003
- **External Systems**: FastAPI gateway

---

### Scope

- **In Scope** :
  - Keep `run_id` stable while allowing a new `thread_id` for a rerun
  - Surface the public identity in the mock API response

- **Out of Scope**:
  - New `thread_id` generation rules
  - Production retry orchestration
  - UI behavior for retry controls

- **Max Increment**: One stable run identity response

---

### Approach

1. Add a mock rerun path that preserves `run_id`
2. Keep the mock flow aligned with the request lifecycle docs

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/request-lifecycle.md` — source of truth for rerun identity behavior
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-101-FUNC.md` — verify `run_id` stays stable
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-101-DOC.md` — verify the stable-identity rule stays documented

---

### Expected Output

- **Deliverable** : Working mock stable run identity response
- **Format** : App code + documentation update
- **Example** : same `run_id` returned on rerun

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-RUNTIME-101-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-RUNTIME-101-DOC.md]]

---

### References

[[../../runtime/request-lifecycle.md]] - Contains the run/thread split and the retry history note that this task must preserve.
