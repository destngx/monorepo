# GW-WF-001: Lock workflow JSON schema and node contracts

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : Critical
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Workflow definition format
- **Feature** : Define canonical workflow JSON schema and node/edge contracts
- **Rationale** : Schema is the user-facing interface; must be locked before MOCK to prevent schema thrashing during implementation

---

### Input

- **Data / Files** : `[[../specification/workflow-schema/WORKFLOW_JSON_SPEC.md]]`, `[[../specification/workflow-schema/README.md]]`, `[[../specification/workflow-schema/plan/schema-and-node-contracts.md]]`, `[[../specification/runtime/universal-interpreter.md]]`
- **Dependencies** : GW-RUNTIME-001 (interpreter interface defines what compiled graphs need)
- **External Systems**: LangGraph (external tool that schema must support)

---

### Scope

- **In Scope** :
  - Define workflow JSON schema (structure, required fields)
  - Define node contract (inputs, outputs, handler, metadata)
  - Define edge contract (source, target, condition)
  - Document graph metadata (version, author, tags)
  - Define validation rules (required nodes, entry/exit points)
  - Define schema versioning for forward/backward compatibility

- **Out of Scope**:
  - Workflow editor or UI
  - Workflow execution semantics (see GW-RUNTIME-\*)
  - Schema migration tooling
  - Example workflows (see GW-SKILL-\*)

- **Max Increment**: Complete workflow schema specification

---

### Approach

1. Synthesize WORKFLOW_JSON_SPEC, schema-and-node-contracts, and runtime interpreter docs
2. Define workflow structure: metadata, nodes, edges
3. Define node contract: type, handler, inputs, outputs, retries
4. Define edge contract: source, target, condition, label
5. Document validation rules and schema versioning

**Files to Modify/Create**:

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` — Confirm JSON schema with examples
- `docs/graph-weave/specification/workflow-schema/plan/schema-and-node-contracts.md` — Confirm contract decisions

---

### Expected Output

- **Deliverable** : Workflow JSON schema specification with validation rules
- **Format** : Markdown with JSON schema and examples
- **Example** :

```json
{
  "version": "1.0",
  "metadata": {
    "name": "order-processing",
    "author": "team",
    "tags": ["e-commerce"]
  },
  "nodes": [
    {
      "id": "validate",
      "type": "python",
      "handler": "validators.check_order",
      "inputs": ["order"],
      "outputs": ["valid_order", "error"]
    }
  ],
  "edges": [
    {
      "source": "validate",
      "target": "process",
      "condition": "output == 'valid_order'"
    }
  ]
}
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-WF-001.md]]`

---

### References

- `[[../specification/workflow-schema/WORKFLOW_JSON_SPEC.md]]` — Canonical schema spec; ensure alignment
- `[[../specification/workflow-schema/plan/schema-and-node-contracts.md]]` — Schema decisions; must be reflected
- `[[./GW-RUNTIME-001.md]]` — Interpreter receives compiled graph; schema must compile to it
- `[[../specification/workflow-schema/MIGRATION_GUIDE.md]]` — Schema versioning; cross-reference migration rules
