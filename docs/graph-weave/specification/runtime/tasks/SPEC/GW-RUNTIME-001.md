# GW-RUNTIME-001: Specify universal workflow interpreter interface

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : LangGraph execution runtime
- **Feature** : Define interpreter contract and input/output boundaries
- **Rationale** : Core abstraction that decouples workflow definition from execution; must be locked before MOCK to prevent implementation churn

---

### Input

- **Data / Files** : `[[../specification/runtime/universal-interpreter.md]]`, `[[../specification/runtime/plan/universal-interpreter-and-skill-loading.md]]`, `[[../specification/runtime/README.md]]`
- **Dependencies** : GW-ARCH-003 (request lifecycle must be defined)
- **External Systems**: LangGraph (external tool boundary)

---

### Scope

- **In Scope** :
  - Define interpreter input contract (compiled graph, state, context, tenant scope)
  - Define interpreter output contract (result, state deltas, events)
  - Document compiled graph structure and node/edge format
  - Define how skills are passed to interpreter (lazy loading signal)
  - Define error signals and recovery points
  - Map interpreter to LangGraph node structure

- **Out of Scope**:
  - LangGraph node implementation code
  - Skill loading logic (see GW-SKILL-\*)
  - Circuit breaker logic (see GW-RUNTIME-\*)
  - Checkpoint implementation (see GW-DATA-\*)

- **Max Increment**: Complete interpreter interface specification

---

### Approach

1. Synthesize universal-interpreter and plan docs
2. Document interpreter as state machine: (graph, state, context) → (result, deltas, events)
3. Define compiled graph schema (nodes, edges, metadata)
4. Define input/output event types
5. Document skill loading boundaries (interpreter expects skills to be provided, not fetches them)

**Files to Modify/Create**:

- `docs/graph-weave/specification/runtime/universal-interpreter.md` — Update with complete interface contract
- `docs/graph-weave/specification/runtime/plan/universal-interpreter-and-skill-loading.md` — Confirm interface decisions

---

### Expected Output

- **Deliverable** : Universal interpreter interface specification
- **Format** : Markdown with TypeScript-like pseudo-code or JSON schema
- **Example** :

```typescript
interface InterpreterInput {
  compiled_graph: CompiledGraph;
  initial_state: ExecutionState;
  context: ExecutionContext; // tenant_id, workflow_id, run_id, etc.
  available_skills: Skill[]; // pre-loaded, not fetched
}

interface InterpreterOutput {
  final_state: ExecutionState;
  result: any;
  events: InterpreterEvent[];
  skill_requests: SkillLoadRequest[]; // for lazy loading in next iteration
}
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-RUNTIME-001.md]]`

---

### References

- `[[../specification/runtime/universal-interpreter.md]]` — Canonical interpreter spec; ensure alignment
- `[[../specification/runtime/plan/universal-interpreter-and-skill-loading.md]]` — Decisions on interpreter scope; must be reflected
- `[[../specification/runtime/request-lifecycle.md]]` — Interpreter sits inside request flow; cross-reference lifecycle
