# GW-RUNTIME-103: Add mock execution endpoint

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime control flow
- **Feature** : Mock workflow execution endpoint
- **Rationale** : MOCK needs one core execution path so the app can accept and respond to a workflow request

---

### Input

- **Data / Files** : `[[../../runtime/request-lifecycle.md]]`, `[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]]`, `[[../../runtime/README.md]]`
- **Dependencies** : GW-ARCH-103
- **External Systems**: FastAPI gateway, mocked LangGraph execution

---

### Scope

- **In Scope** :
  - Add the mock workflow execution endpoint
  - Parse the request and return a mocked workflow response

- **Out of Scope**:
  - Real LangGraph execution
  - Production retry orchestration
  - UI flow changes

- **Max Increment**: One runnable mock execution endpoint

---

### Approach

1. Add the execution endpoint to the mock app
2. Return a deterministic mocked workflow response

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/request-lifecycle.md` — source of truth for request flow
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-103-FUNC.md` — verify the endpoint returns a mock workflow response
- `docs/graph-weave/specification/runtime/verification/MOCK/VERIFY-GW-RUNTIME-103-DOC.md` — verify the execution rule stays documented

---

### Expected Output

- **Deliverable** : Working mock execution endpoint
- **Format** : REST endpoint + documentation update
- **Example** : request in, mocked workflow response out

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-RUNTIME-103-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-RUNTIME-103-DOC.md]]

---

### References

[[../../runtime/request-lifecycle.md]] - Defines the two-step request lifecycle that the mock endpoint must follow.
