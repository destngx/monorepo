# GW-ARCH-102: Add dynamic OpenAPI docs

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Platform architecture
- **Feature** : Dynamic Swagger/OpenAPI docs in the mock app
- **Rationale** : MOCK needs one generated API documentation surface so params, body, and descriptions are visible

---

### Input

- **Data / Files** : `[[../../system-architecture.md]]`, `[[../../macro-architecture.md]]`, `[[../../plan/platform-boundary-and-fixed-stack.md]]`
- **Dependencies** : GW-ARCH-101
- **External Systems**: FastAPI, Swagger/OpenAPI generator

---

### Scope

- **In Scope** :
  - Implement dynamic Swagger/OpenAPI output in the mock app
  - Include params, request body, and descriptions in the generated docs

- **Out of Scope**:
  - Logging behavior
  - Production observability pipelines
  - Deployment manifests

- **Max Increment**: One working mock API docs path

---

### Approach

1. Wire the OpenAPI generator into the mock app
2. Ensure params, body, and descriptions are rendered dynamically

**Files to Modify/Create**:

- `docs/graph-weave/specification/architecture/system-architecture.md` — source of truth for OpenAPI behavior
- `docs/graph-weave/specification/architecture/verification/MOCK/VERIFY-GW-ARCH-102-FUNC.md` — verify generated docs fields exist
- `docs/graph-weave/specification/architecture/verification/MOCK/VERIFY-GW-ARCH-102-DOC.md` — verify the OpenAPI requirement stays documented

---

### Expected Output

- **Deliverable** : Working mock API docs path
- **Format** : App code + documentation update
- **Example** : generated docs show params, body, and descriptions

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-ARCH-102-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-ARCH-102-DOC.md]]

---

### References

[[../../system-architecture.md]] - Contains the base platform requirements and is the source of truth for OpenAPI expectations.
