# GW-DATA-002: Specify checkpoint and compiled graph storage

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : TBD

---

### Context

- **Bounded Context** : Execution state persistence
- **Feature** : Define checkpoint structure and resume semantics
- **Rationale** : Checkpoint strategy affects availability and resume guarantees; must be locked before MOCK to prevent replay bugs

---

### Input

- **Data / Files** : `[[../specification/data/plan/compiled-graph-and-checkpoint-storage.md]]`, `[[../specification/data/README.md]]`, `[[../specification/runtime/universal-interpreter.md]]`
- **Dependencies** : GW-DATA-001 (Redis namespace), GW-RUNTIME-001 (interpreter interface)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define checkpoint structure (what state to persist and when)
  - Define compiled graph caching strategy (cache vs. fetch vs. embed)
  - Document resume semantics (how to restore from checkpoint)
  - Define TTL for checkpoints vs. cached graphs
  - Document what is checkpoint-safe vs. ephemeral
  - Map storage to interpreter state contract

- **Out of Scope**:
  - Compiled graph generation logic (workflow schema concern)
  - Actual graph serialization format (value encoding)
  - Database schema
  - Cleanup and retention policies (ops concern)

- **Max Increment**: Complete checkpoint and cache specification

---

### Approach

1. Synthesize compiled-graph-and-checkpoint-storage plan
2. Define checkpoint structure: {run_id, last_node, state, timestamp}
3. Define what is captured: interpreter state, event log, execution context
4. Define resume behavior: restore state, continue from last_node
5. Define cache strategy: compiled graphs cached in Redis or re-fetched from registry

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/plan/compiled-graph-and-checkpoint-storage.md` — Confirm checkpoint structure and cache decisions

---

### Expected Output

- **Deliverable** : Checkpoint and graph cache specification
- **Format** : Markdown with JSON schema and state diagram
- **Example** :

```json
{
  "checkpoint": {
    "run_id": "r123",
    "version": 1,
    "last_node": "n5",
    "timestamp": "2024-04-08T12:00:00Z",
    "state": {
      "context": {...},
      "messages": [...],
      "node_state": {...}
    }
  },
  "cache_strategy": "Redis-backed with 7-day TTL"
}
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-DATA-002.md]]`

---

### References

- `[[../specification/data/plan/compiled-graph-and-checkpoint-storage.md]]` — Checkpoint decisions; must be reflected in spec
- `[[../specification/runtime/universal-interpreter.md]]` — Interpreter state contract; ensure checkpoint captures all needed fields
- `[[../specification/workflow-schema/WORKFLOW_JSON_SPEC.md]]` — Workflow definition format; ensure graph is cacheable
