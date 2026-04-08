# GW-ARCH-003: Lock runtime execution flow contract

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Runtime architecture
- **Feature** : Define two-request lifecycle and SSE streaming
- **Rationale** : Client-facing contract must be locked before implementation; determines checkpoint strategy, error recovery, and streaming semantics

---

### Input

- **Data / Files** : `[[../specification/runtime/request-lifecycle.md]]`, `[[../specification/runtime/plan/request-lifecycle-and-streaming.md]]`, `[[../specification/runtime/README.md]]`
- **Dependencies** : GW-ARCH-001 (system boundaries), GW-ARCH-002 (tenant isolation)
- **External Systems**: None (defines contract, not implementation)

---

### Scope

- **In Scope** :
  - Document two-request lifecycle: `POST /execute` (submit) + `GET /execute/{run_id}/status` (stream)
  - Define SSE event types and structure (request, node, tool, checkpoint, completion)
  - Define checkpoint persistence and resume semantics
  - Document validation boundary (when and where requests are rejected)
  - Define active-thread cleanup rules on completion
  - Map lifecycle to Redis state transitions

- **Out of Scope**:
  - LangGraph node implementation
  - SSE library selection or encoding details
  - Redis key structure (see GW-DATA-\*)
  - Guardrails/circuit-breaker behavior (see GW-RUNTIME-\*)

- **Max Increment**: Complete request lifecycle specification with event contracts

---

### Approach

1. Synthesize request-lifecycle and request-lifecycle-and-streaming docs
2. Document submit request: what is accepted, what validation occurs, what is returned
3. Document status request: what SSE events are emitted, in what order, when
4. Define checkpoint structure and resume behavior
5. Define completion behavior (what happens to run state)

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/request-lifecycle.md` — Confirm two-request flow, event types, validation points
- `docs/graph-weave/specification/runtime/plan/request-lifecycle-and-streaming.md` — Ensure decisions documented

---

### Expected Output

- **Deliverable** : Request lifecycle and SSE contract specification
- **Format** : Markdown with state diagrams and JSON schema examples
- **Example** :

```json
// Submit request (POST /execute)
{
  "tenant_id": "t123",
  "workflow_id": "w456",
  "inputs": {...}
}

// Response
{
  "run_id": "r789"
}

// SSE event (GET /execute/{run_id}/status)
event: checkpoint
data: {"run_id": "r789", "last_node": "n1", "state": {...}}
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-ARCH-003.md]]`

---

### References

- `[[../specification/runtime/request-lifecycle.md]]` — Canonical lifecycle doc; ensure alignment
- `[[../specification/runtime/plan/request-lifecycle-and-streaming.md]]` — Decisions on streaming; must be implemented
- `[[../specification/runtime/README.md]]` — Runtime component scope; ensure within boundaries
