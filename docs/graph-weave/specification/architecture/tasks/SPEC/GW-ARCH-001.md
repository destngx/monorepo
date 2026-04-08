# GW-ARCH-001: Document system architecture boundaries

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Platform architecture
- **Feature** : Establish and document fixed stack contract
- **Rationale** : Must lock system boundaries before implementation to prevent architecture drift and ensure LangGraph/FastAPI/Redis/MCP alignment

---

### Input

- **Data / Files** : `[[../specification/architecture/system-architecture.md]]`, `[[../specification/architecture/macro-architecture.md]]`, `[[../specification/architecture/plan/platform-boundary-and-fixed-stack.md]]`
- **Dependencies** : None (foundational)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Document fixed platform stack: FastAPI, LangGraph, Redis, MCP
  - Define request flow: gateway → runtime → tools
  - Define API contract boundaries (what clients see)
  - Map macro layers (API gateway, interpreter, storage, external tools)

- **Out of Scope**:
  - Implementation code
  - Detailed Redis key design (see GW-DATA-\*)
  - Tenant isolation logic (see GW-ARCH-003)
  - Skill loading specifics (see GW-SKILL-\*)

- **Max Increment**: Single document defining system context and boundaries

---

### Approach

1. Synthesize existing spec docs into unified system context diagram
2. Document three-layer architecture: Gateway (stateless FastAPI) → Runtime (LangGraph + Redis) → External (MCP tools)
3. Define immutable platform contracts for each layer
4. Cross-reference to macro-architecture and system-architecture specs

**Files to Modify/Create**:

- `docs/graph-weave/specification/architecture/system-architecture.md` — Update or confirm system context and layers (ensure 1-page overview)
- `docs/graph-weave/specification/architecture/macro-architecture.md` — Verify macro layer descriptions against system context

---

### Expected Output

- **Deliverable** : System architecture document with clear layer boundaries, platform stack confirmation, and external API contract
- **Format** : Markdown with ASCII diagrams or Mermaid if helpful
- **Example** :

```
FastAPI Gateway
  ↓ (stateless submission)
LangGraph Runtime (Redis-backed)
  ↓ (tool invocation)
MCP External Tools
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-ARCH-001.md]]`

---

### References

- `[[../specification/architecture/system-architecture.md]]` — Canonical system context; ensure alignment
- `[[../specification/architecture/macro-architecture.md]]` — Macro layers; ensure consistent terminology
- `[[../specification/architecture/plan/platform-boundary-and-fixed-stack.md]]` — Fixed stack decisions; must be reflected in task output
