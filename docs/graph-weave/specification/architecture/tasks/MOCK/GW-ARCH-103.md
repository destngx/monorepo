# GW-ARCH-103: Bootstrap mock application core

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Platform architecture
- **Feature** : Runnable mock application core
- **Rationale** : MOCK needs one executable app entrypoint before the feature slices can be exercised together

---

### Input

- **Data / Files** : `[[../../system-architecture.md]]`, `[[../../macro-architecture.md]]`, `[[../../plan/platform-boundary-and-fixed-stack.md]]`
- **Dependencies** : None
- **External Systems**: FastAPI

---

### Scope

- **In Scope** :
  - Create the mock app entrypoint and basic routing shell
  - Wire the app so the other MOCK behaviors can attach to it

- **Out of Scope**:
  - Business logic endpoints
  - Production deployment manifests
  - Multi-tenant isolation

- **Max Increment**: One runnable mock app core

---

### Approach

1. Create the FastAPI app shell
2. Add the minimal route structure needed by the mock tasks

**Files to Modify/Create**:

- `docs/graph-weave/specification/architecture/system-architecture.md` — source of truth for the app shape
- `docs/graph-weave/specification/architecture/verification/MOCK/VERIFY-GW-ARCH-103-FUNC.md` — verify the app shell exists
- `docs/graph-weave/specification/architecture/verification/MOCK/VERIFY-GW-ARCH-103-DOC.md` — verify the bootstrapping rule stays documented

---

### Expected Output

- **Deliverable** : Working mock application core
- **Format** : App code + documentation update
- **Example** : app entrypoint with router registration

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-ARCH-103-FUNC.md]]
[[../../verification/MOCK/VERIFY-GW-ARCH-103-DOC.md]]

---

### References

[[../../system-architecture.md]] - Defines the overall platform shape and the FastAPI gateway boundary.
