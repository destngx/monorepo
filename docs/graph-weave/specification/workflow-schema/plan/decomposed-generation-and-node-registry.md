# Plan: Reusable Node Library and Composable Workflow Generation

**Date**: 2026-05-14
**Status**: Proposed
**Scope**: Breaking refactor — node registry + compositional workflows

---

## 1. Problem Statement

### Current Issues

| Issue                                            | Impact                                                    |
| ------------------------------------------------ | --------------------------------------------------------- |
| LLM generates full workflow JSON in one pass     | Non-deterministic results, especially for complex intents |
| Nodes embedded inline in workflow JSON           | No reuse across workflows, duplication                    |
| Entire workflow versioned as single unit         | Can't version nodes independently                         |
| Debugging requires full workflow execution       | Hard to isolate node-level issues                         |
| No node catalog                                  | Can't browse or discover existing nodes                   |
| Generator regenerates same node types repeatedly | Wasteful, inconsistent                                    |

### Root Cause Analysis (5 Whys)

1. **Why inconsistent?** → LLM generates complete workflow JSON with all nodes, edges, prompts at once
2. **Why is that a problem?** → Too many interconnected variables to track in single generation pass
3. **Why too many?** → Node IDs, edge references, prompt alignment, output schemas, state paths must all cohere
4. **Why does coherence fail?** → Combinatorial complexity exceeds LLM working memory
5. **Why can't we fix prompts?** → No prompt engineering overcomes fundamental constraint: more variables = higher error probability

**Real Need**: Reduce constraint surface per LLM call by decomposing generation into atomic, independently-verifiable steps.

---

## 2. Solution Overview

### Two Complementary Improvements

```
┌─────────────────────────────────────────────────────────────┐
│                    IMPROVEMENT 1                             │
│              Decomposed Generation Pipeline                  │
├─────────────────────────────────────────────────────────────┤
│  Current: planner → node_builder → edge_router → assembler  │
│  Proposed: planner → node_selector → edge_generator → assembler│
│                                                             │
│  Key change: SELECT nodes from registry instead of GENERATE │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    IMPROVEMENT 2                             │
│                  Node Registry                               │
├─────────────────────────────────────────────────────────────┤
│  Current: Nodes embedded in workflow JSON                   │
│  Proposed: Nodes stored independently in Redis, referenced  │
└─────────────────────────────────────────────────────────────┘
```

### Combined Effect

| Current                                     | Proposed                                       |
| ------------------------------------------- | ---------------------------------------------- |
| Generate 500+ word system prompts each time | Select `research-analyst:v1.2.0` from registry |
| Generate schemas, guardrails each time      | Load from node definition                      |
| Debug by running full workflow              | Test node in isolation                         |
| No reuse across workflows                   | Same node in 10+ workflows                     |
| Version entire workflow                     | Version nodes independently                    |

---

## 3. Decisions

### Core Model

- Nodes become first-class tenant-scoped resources stored in Redis as the primary source of truth.
- Each node version has a distinct immutable node ID, for example `normalize_input:v1.0.0`.
- Node ID format: `{node_name}:v{major}.{minor}.{patch}` (colon-separated name and version).
- Workflows store the compositional source format (node references), never the compiled form.
- Runtime compiles compositional workflows to embedded form before execution.
- No legacy embedded format support — this is a breaking refactor.

### Node Immutability

- **Once created, a node ID's `config`, `input_contract`, and `output_contract` are immutable.**
- PUT `/nodes/{node_id}` may only update `name`, `description`, `tags` (display metadata).
- To change executable fields, create a new node ID with bumped version.
- This guarantees workflows pinned to a node ID always execute the same logic.

### Generated Node Safeguards

- Generated custom nodes are automatically saved as active nodes in the tenant registry.
- **All generated nodes must pass validation before save** (schema, contracts, standalone executability).
- Generated nodes include provenance metadata:
  ```json
  {
    "provenance": {
      "generated_by_workflow": "workflow-generator:v2.0.0",
      "generated_by_run_id": "run_abc123",
      "generated_at": "2026-05-14T10:30:00Z",
      "source_intent": "Research AI ethics..."
    },
    "reuse_eligible": true
  }
  ```
- `reuse_eligible` defaults to `true` for validated nodes, `false` for nodes that passed validation but are flagged as potentially one-off.
- Nodes with `reuse_eligible: false` are still usable by the generating workflow but excluded from registry search results.

### Tenant Isolation

- No global/shared node library — tenant-scoped only.
- Nodes are always public within tenant (no private nodes).
- Tenant ID is required for every node API operation.
- Semantic duplicate detection may suggest reuse, but persisted workflows only store exact node IDs.

### Persistence Model

- **Redis stores the compositional source only** (node references, not embedded nodes).
- Key: `workflows:{tenant_id}:{workflow_id}` → compositional workflow JSON.
- At execution time, the runtime compiler resolves node IDs → embedded nodes.
- This ensures:
  - Reproducibility: source is always the source of truth.
  - Debuggability: can inspect which node versions a workflow uses.
  - Recompilability: can recompile if node metadata changes (but not config).

---

## 4. Schemas

### Compositional Workflow Source Schema

```json
{
  "$id": "graphweave/compositional-workflow/v1",
  "type": "object",
  "required": ["name", "version", "nodes", "edges"],
  "properties": {
    "name": { "type": "string" },
    "version": { "type": "string" },
    "description": { "type": "string" },
    "metadata": {
      "type": "object",
      "properties": {
        "author": { "type": "string" },
        "tenant_id": { "type": "string" },
        "created_at": { "type": "string", "format": "date-time" },
        "tags": { "type": "array", "items": { "type": "string" } }
      }
    },
    "limits": {
      "type": "object",
      "properties": {
        "max_hops": { "type": "integer", "default": 20 },
        "max_tokens": { "type": "integer", "default": 100000 },
        "timeout_seconds": { "type": "integer", "default": 300 }
      }
    },
    "nodes": {
      "type": "array",
      "items": {
        "oneOf": [
          { "$ref": "#/$defs/entry_node" },
          { "$ref": "#/$defs/exit_node" },
          { "$ref": "#/$defs/node_reference" }
        ]
      }
    },
    "edges": {
      "type": "array",
      "items": { "$ref": "#/$defs/edge" }
    }
  },
  "$defs": {
    "entry_node": {
      "type": "object",
      "required": ["id", "type"],
      "properties": {
        "id": { "const": "entry" },
        "type": { "const": "entry" },
        "config": {
          "type": "object",
          "properties": {
            "properties": { "type": "object" },
            "required": { "type": "array", "items": { "type": "string" } }
          }
        }
      }
    },
    "exit_node": {
      "type": "object",
      "required": ["id", "type"],
      "properties": {
        "id": { "const": "exit" },
        "type": { "const": "exit" },
        "config": {
          "type": "object",
          "properties": {
            "output_mapping": { "type": "object" }
          }
        }
      }
    },
    "node_reference": {
      "type": "object",
      "required": ["alias", "node_id"],
      "properties": {
        "alias": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*$",
          "description": "Workflow-local identifier for this node instance"
        },
        "node_id": {
          "type": "string",
          "pattern": "^[a-z][a-z0-9_]*:v[0-9]+\\.[0-9]+\\.[0-9]+$",
          "description": "Exact node ID in registry (name:version)"
        },
        "overrides": {
          "type": "object",
          "description": "Optional workflow-specific overrides (input_mapping, output_key only)",
          "properties": {
            "input_mapping": { "type": "object" },
            "output_key": { "type": "string" }
          },
          "additionalProperties": false
        }
      }
    },
    "edge": {
      "type": "object",
      "required": ["from", "to"],
      "properties": {
        "from": { "type": "string" },
        "to": { "type": "string" },
        "condition": { "type": "string" },
        "config": {
          "type": "object",
          "properties": {
            "label": { "type": "string" }
          }
        }
      }
    }
  }
}
```

### Compiled Workflow Schema (for execution)

The compiled workflow follows the existing `WORKFLOW_JSON_SPEC.md` embedded format:

```json
{
  "name": "...",
  "version": "...",
  "nodes": [
    {
      "id": "entry",
      "type": "entry",
      "config": { ... }
    },
    {
      "id": "normalize_input",
      "type": "agent_node",
      "display_name": "Normalize Input",
      "description": "...",
      "config": {
        "system_prompt": "...",
        "user_prompt_template": "...",
        "input_mapping": { ... },
        "output_schema": { ... }
      },
      "guardrails": { ... },
      "retry_config": { ... }
    },
    {
      "id": "exit",
      "type": "exit",
      "config": { ... }
    }
  ],
  "edges": [
    { "from": "entry", "to": "normalize_input" },
    { "from": "normalize_input", "to": "exit" }
  ]
}
```

**Compilation rules**:

1. `entry` and `exit` nodes pass through unchanged.
2. Node references (`alias`, `node_id`) are resolved from registry.
3. `alias` becomes the compiled node's `id`.
4. Node's `config`, `guardrails`, `retry_config` are loaded from registry.
5. `overrides.input_mapping` and `overrides.output_key` are merged into the loaded config.

### Node Schema (stored in Redis)

```json
{
  "tenant_id": "pkm",
  "node_id": "normalize_input:v1.0.0",
  "node_name": "normalize_input",
  "version": "1.0.0",
  "name": "Normalize Input",
  "type": "agent_node",
  "description": "Trim content and normalize tag input",
  "config": {
    "system_prompt": "...",
    "user_prompt_template": "...",
    "input_mapping": {},
    "output_schema": {}
  },
  "input_contract": {
    "required": [
      { "name": "file_path", "type": "string" },
      { "name": "file_content", "type": "string" }
    ],
    "optional": [{ "name": "existing_tags", "type": "array" }]
  },
  "output_contract": {
    "produced": [
      { "name": "file_path", "type": "string", "state_path": "$.output.file_path" },
      { "name": "file_content", "type": "string", "state_path": "$.output.file_content" },
      { "name": "existing_tags", "type": "array", "state_path": "$.output.existing_tags" }
    ]
  },
  "tags": ["pkm", "normalization"],
  "owner": "system",
  "status": "active",
  "reuse_eligible": true,
  "provenance": {
    "generated_by_workflow": null,
    "generated_by_run_id": null,
    "generated_at": "2026-05-14T10:30:00Z",
    "source_intent": null
  },
  "created_at": "2026-05-14T10:30:00Z",
  "immutable_fields": ["config", "input_contract", "output_contract", "type"]
}
```

**Contract format changes**:

- `input_contract.required` and `input_contract.optional` now include `type` alongside `name`.
- `output_contract.produced` replaces `output_contract.required`, with `state_path` for runtime resolution.
- This enables type-aware contract validation, not just field-name matching.

---

## 5. Validation Rules

### Node Validation

- A stored node must be executable without depending on a specific workflow.
- A stored node must declare required inputs (with types) and produced outputs (with state paths).
- A node's `config`, `input_contract`, `output_contract`, and `type` are immutable after creation.
- PUT may only update `name`, `description`, `tags`.
- All generated nodes must pass standalone validation before save.

### Workflow Validation

- A workflow must not reference unresolved node IDs.
- All node references must use `alias` (workflow-local) and `node_id` (registry exact).
- Edges must reference aliases, not node IDs.
- Edge `from` must be `"entry"` or a defined alias.
- Edge `to` must be `"exit"` or a defined alias.
- No cycles allowed (validated at compile time).

### Contract Validation (at compile time)

- For each edge `A → B`, verify:
  - Every field in `B.input_contract.required` exists in `A.output_contract.produced` (by `name`).
  - Types match between upstream `produced` and downstream `required`.
  - `state_path` in `produced` is a valid JSONPath expression.
- Reject incompatible chains with specific error messages.

### Redis Key Design

To avoid parsing issues with `:` in node IDs:

```
nodes:{tenant_id}:{node_id}              → Full node JSON (HSET)
nodes:{tenant_id}:index                  → SET of node_ids (for discovery)
nodes:{tenant_id}:name:{node_name}       → SET of node_ids for this name
nodes:{tenant_id}:tags:{tag}             → SET of node_ids with this tag
```

- Use Redis HSET for node storage, not string keys that require parsing.
- Indexes are SET structures containing node_id values.
- Never parse node_id from Redis key — always read from stored JSON.

---

## 6. Generator Flow

1. Plan ordered workflow steps from the user intent.
2. For each step:
   a. Search tenant node registry for a compatible reusable node.
   b. If match found with compatible contract → use existing node_id.
   c. If no match → generate custom node, validate, auto-save, use new node_id.
3. Assemble compositional workflow with aliases and node_ids.
4. Generate edges between aliases (with contract validation).
5. Validate final topology (DAG, no orphans, entry/exit connected).
6. Return compositional workflow JSON (source form, not compiled).

---

## 7. API Design

### Node CRUD

```
POST   /nodes                              # Create node; tenant_id in body
GET    /nodes?tenant_id=...&tags=...        # List/search nodes
GET    /nodes/{node_id}?tenant_id=...       # Get specific node ID
PUT    /nodes/{node_id}?tenant_id=...       # Update metadata ONLY (name, description, tags)
DELETE /nodes/{node_id}?tenant_id=...       # Delete node ID
```

### Workflow CRUD (modified)

```
POST   /workflows                           # Create workflow (compositional format)
GET    /workflows/{workflow_id}?tenant_id=.. # Get workflow (returns compositional source)
PUT    /workflows/{workflow_id}?tenant_id=.. # Update workflow
DELETE /workflows/{workflow_id}?tenant_id=.. # Delete workflow
```

### Execution (modified)

```
POST   /execute                             # Execute workflow
                                              # Runtime compiles node_ids → embedded before execution
```

---

## 8. Redis Storage

### Node Keys

```
nodes:{tenant_id}:{node_id}              → HSET: full node JSON
nodes:{tenant_id}:index                  → SET: all node_ids in tenant
nodes:{tenant_id}:name:{node_name}       → SET: node_ids for this name
nodes:{tenant_id}:tags:{tag}             → SET: node_ids with this tag
```

### Workflow Keys

```
workflows:{tenant_id}:{workflow_id}       → compositional workflow JSON
workflows:{tenant_id}:index               → SET: workflow_ids in tenant
```

### No Compiled Workflow Storage

Compiled workflows are never persisted. Compilation happens at:

1. Execution time (runtime compiler)
2. Explicit compile API call (for debugging)

---

## 9. Implementation Roadmap

### Phase 1: Node Registry Infrastructure

| Task       | Priority | Effort | Description                                        |
| ---------- | -------- | ------ | -------------------------------------------------- |
| GW-REG-001 | High     | 3d     | Node models (create, update, response, validators) |
| GW-REG-002 | High     | 2d     | Node Redis store with typed contracts              |
| GW-REG-003 | High     | 2d     | Node validation (schema, contracts, immutability)  |
| GW-REG-004 | High     | 2d     | Node CRUD API endpoints                            |
| GW-REG-005 | Medium   | 1d     | Node discovery/search by tags                      |

**Total**: ~10 days

### Phase 2: Seed Node Library + Runtime Compiler

| Task       | Priority | Effort | Description                                                                |
| ---------- | -------- | ------ | -------------------------------------------------------------------------- |
| GW-LIB-001 | High     | 1d     | Extract `normalize_input` node                                             |
| GW-LIB-002 | High     | 1d     | Extract `classify_metadata` node                                           |
| GW-LIB-003 | High     | 2d     | Extract 5+ common nodes (research, summarize, verify, classify, transform) |
| GW-RUN-001 | High     | 2d     | Workflow compiler: resolve node_ids → embedded                             |
| GW-RUN-002 | High     | 1d     | Contract validation (type-aware)                                           |
| GW-RUN-003 | High     | 2d     | Parser: compositional format only (remove embedded support)                |
| GW-RUN-004 | High     | 1d     | Update builder to use compiled workflows                                   |

**Total**: ~10 days

### Phase 3: Generator v2.0.0

| Task       | Priority | Effort | Description                                        |
| ---------- | -------- | ------ | -------------------------------------------------- |
| GW-GEN-001 | High     | 2d     | Create `workflow-generator:v2.0.0.json`            |
| GW-GEN-002 | High     | 3d     | Node selector tools (search, create, find similar) |
| GW-GEN-003 | Medium   | 2d     | Edge generator with contract validation            |
| GW-GEN-004 | Medium   | 2d     | Custom node generation with provenance             |
| GW-GEN-005 | Low      | 2d     | Comparison tests (v1 vs v2 consistency)            |

**Total**: ~11 days

### Phase 4: Cleanup & Documentation

| Task         | Priority | Effort | Description                                            |
| ------------ | -------- | ------ | ------------------------------------------------------ |
| GW-CLEAN-001 | High     | 1d     | Remove old workflow JSON spec references               |
| GW-CLEAN-002 | Medium   | 1d     | Update WORKFLOW_JSON_SPEC.md with compositional schema |
| GW-CLEAN-003 | Low      | 1d     | Node authoring guide                                   |

**Total**: ~3 days

---

## 10. Testing Strategy

### Unit Tests

```python
# Test node CRUD
async def test_create_node():
    """Node creation rejects missing config, missing output schema, invalid type"""
    with pytest.raises(ValidationError):
        await create_node({"node_id": "test"})  # Missing required fields

async def test_node_immutability():
    """PUT rejects changes to config, input_contract, output_contract"""
    await create_node({"node_id": "test:v1.0.0", "config": {...}, ...})
    with pytest.raises(ImmutableFieldError):
        await update_node("test:v1.0.0", {"config": {"system_prompt": "new"}})

async def test_tenant_isolation():
    """Tenant A cannot read or reuse Tenant B node definitions"""
    await create_node({"tenant_id": "a", "node_id": "test:v1.0.0", ...})
    with pytest.raises(NotFound):
        await get_node("test:v1.0.0", tenant_id="b")

# Test contract validation (type-aware)
async def test_contract_compatibility():
    """Edge generation rejects incompatible node chains (type mismatch)"""
    node_a = {"output_contract": {"produced": [{"name": "data", "type": "object"}]}}
    node_b = {"input_contract": {"required": [{"name": "data", "type": "string"}]}}
    with pytest.raises(IncompatibleContract):
        validate_edge(node_a, node_b)
```

### Integration Tests

```python
# Test workflow compilation
async def test_compile_node_ids_to_embedded():
    """Workflow compilation resolves exact node IDs into embedded executable nodes"""
    workflow = {
        "nodes": [
            {"alias": "normalize", "node_id": "normalize_input:v1.0.0"}
        ],
        "edges": [...]
    }
    compiled = await compile_workflow(workflow)
    assert "config" in compiled["nodes"][0]  # Now embedded
    assert compiled["nodes"][0]["config"]["system_prompt"] != ""

# Test version pinning
async def test_version_pinning():
    """Creating node:v1.0.1 does not alter workflows pinned to v1.0.0"""
    await create_node({"node_id": "test:v1.0.0", ...})
    workflow = {"nodes": [{"alias": "a", "node_id": "test:v1.0.0"}], ...}

    await create_node({"node_id": "test:v1.0.1", ...})
    compiled = await compile_workflow(workflow)
    assert compiled["nodes"][0]["config"]["version"] == "1.0.0"  # Unchanged
```

### E2E Tests

```python
# Test generator v2
async def test_generator_creates_validated_node_ids():
    """Generator creates validated node IDs, edges, and compositional workflow"""
    result = await execute_workflow(
        "workflow-generator:v2.0.0",
        {"intent": "Research AI ethics and summarize findings"}
    )

    generated = result["generated_workflow"]
    assert "node_id" in generated["nodes"][0]

    # Validate edge endpoints
    aliases = {n["alias"] for n in generated["nodes"]}
    valid_from = {"entry"} | aliases
    valid_to = {"exit"} | aliases
    assert all(e["from"] in valid_from for e in generated["edges"])
    assert all(e["to"] in valid_to for e in generated["edges"])
```

### Test Matrix

| Test File                         | Coverage                                                      |
| --------------------------------- | ------------------------------------------------------------- |
| `test_node_store.py`              | Node CRUD, tenant isolation, index maintenance, immutability  |
| `test_node_compiler.py`           | Compilation, contract validation (type-aware), error handling |
| `test_node_api.py`                | API endpoints, validation, error responses                    |
| `test_langgraph_graph_builder.py` | Parser (compositional only), builder with compiled nodes      |
| `test_e2e_node_registry.py`       | Full node lifecycle via API                                   |
| `test_e2e_generator_v2.py`        | Generator v2 end-to-end                                       |
| `test_e2e_workflow_execution.py`  | Execution with compositional workflows                        |

---

## 11. Success Metrics

### Generation Reliability

| Metric                     | Current (v1) | Target (v2) |
| -------------------------- | ------------ | ----------- |
| Valid DAG generation rate  | ~70%         | >95%        |
| Node config completeness   | ~80%         | >99%        |
| Edge reference validity    | ~85%         | >99%        |
| Average generation time    | 45s          | 20s         |
| Token usage per generation | 15k          | 5k          |

### Node Reuse

| Metric                                   | Current | Target (6 months) |
| ---------------------------------------- | ------- | ----------------- |
| Workflows using shared nodes             | 0%      | >60%              |
| Average nodes per workflow from registry | 0       | 3+                |
| Node catalog size                        | 0       | 20+               |
| Node test coverage                       | 0%      | >80%              |

---

## 12. References

- Current generator: `src/resources/workflows/workflow-generator:v1.0.1.json`
- Workflow JSON spec: `specification/workflow-schema/WORKFLOW_JSON_SPEC.md` (to be updated)
- Skills architecture: `specification/skills/llm-skills-architecture.md`

---

**End of Document**
