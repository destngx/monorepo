# Workflow JSON Schema System

Local entrypoint for the prompt-driven workflow schema.

## Use this folder for

- workflow JSON entry docs
- migration notes
- local references to deeper schema files

## Local references

- `[[WORKFLOW_JSON_SPEC.md]]`
- `[[MIGRATION_GUIDE.md]]`
- `[[plan/README]]`
- `[[tasks/README]]`
- `[[verification/README]]`
- `[[../README]]`

---

## How It Works at Runtime

### 1. **Initialization**

- Load workflow JSON.
- Validate schema and structure.
- Build LangGraph with nodes and edges.

### 2. **Execution**

```
Entry Node → [Input Config] → Current State

For each node:
  1. Evaluate incoming edges
  2. Find first edge with true condition (or first unconditional edge)
  3. Execute next node:
   - If agent_node: load skills on demand, execute, store output
     - If branch: evaluate condition, don't execute work
     - If human_decision: pause, wait for user input
     - If exit: return final output
  4. Repeat until exit node reached (or limits exceeded)
```

### 3. **Edge Condition Evaluation**

Conditions are JSONPath expressions:

```python
# In Python/LangGraph runtime:

def should_take_edge(edge, workflow_state):
    if not edge.condition:
        return True  # Unconditional edge

    # Evaluate JSONPath expression
    left_value = get_value_at_path(workflow_state, edge.condition.left_path)
    right_value = parse_literal(edge.condition.right_value)
    operator = edge.condition.operator

    return evaluate(left_value, operator, right_value)
```

### 4. **State Management**

State is immutable during edge evaluation:

```python
# Snapshot state before evaluating edges
state_snapshot = copy.deepcopy(workflow_state)

# Evaluate all outgoing edges
eligible_edges = [
    e for e in outgoing_edges
    if evaluate_condition(e.condition, state_snapshot)
]

# Take first eligible edge (deterministic)
next_node = eligible_edges[0].to
```

---

## Common Patterns

### Pattern 1: Decision Tree

Route to different agents based on classification:

```json
{
  "nodes": [
    { "id": "classify", "type": "agent_node", "config": { "system_prompt": "...", "user_prompt_template": "..." } },
    { "id": "route", "type": "branch", "config": { "condition_expression": "$.classification.type" } },
    {
      "id": "billing_agent",
      "type": "agent_node",
      "config": { "system_prompt": "...", "user_prompt_template": "..." }
    },
    {
      "id": "shipping_agent",
      "type": "agent_node",
      "config": { "system_prompt": "...", "user_prompt_template": "..." }
    }
  ],
  "edges": [
    { "from": "classify", "to": "route" },
    { "from": "route", "to": "billing_agent", "condition": "$.classification.type == 'billing'" },
    { "from": "route", "to": "shipping_agent", "condition": "$.classification.type == 'shipping'" }
  ]
}
```

### Pattern 2: Quality Gate with Fallback

Retry or escalate if confidence is low:

```json
{
  "edges": [
    { "from": "research", "to": "summarize", "condition": "$.research.confidence > 0.8" },
    { "from": "research", "to": "rework", "condition": "$.research.confidence <= 0.8" },
    { "from": "rework", "to": "research" } // Loop back
  ]
}
```

### Pattern 3: Parallel Processing with Merge

Multiple agents, then aggregation:

```json
{
  "nodes": [
    { "id": "split", "type": "branch" },
    { "id": "agent_a", "type": "agent_node", "config": { "system_prompt": "...", "user_prompt_template": "..." } },
    { "id": "agent_b", "type": "agent_node", "config": { "system_prompt": "...", "user_prompt_template": "..." } },
    { "id": "merge", "type": "agent_node", "config": { "system_prompt": "...", "user_prompt_template": "..." } }
  ],
  "edges": [
    { "from": "split", "to": "agent_a" },
    { "from": "split", "to": "agent_b" },
    { "from": "agent_a", "to": "merge" },
    { "from": "agent_b", "to": "merge" }
  ]
}
```

---

## Troubleshooting

| Issue                      | Solution                                                        |
| -------------------------- | --------------------------------------------------------------- |
| **Unreachable node**       | Check incoming edges; all non-entry nodes need incoming path.   |
| **Condition always false** | Verify JSONPath is correct; check state structure.              |
| **Missing prompt fields**  | Add `system_prompt` and `user_prompt_template` to `agent_node`. |
| **Circular loop**          | Ensure loop has explicit stagnation detection in limits.        |
| **Output mapping wrong**   | Verify JSONPath in exit node matches actual state keys.         |

---

## Migration from Old Model

See `MIGRATION_GUIDE.md` for detailed migration steps from subagent-routing workflows.

**Key changes**:

- ❌ Remove top-level `skills` array
- ❌ Remove `subagents` array
- ✅ Add explicit `nodes` array (entry, exit, agent_node, branch)
- ✅ Add explicit `edges` array with conditions
- ✅ Move behavior to node-level `system_prompt` and `user_prompt_template`

---

## References

- **Authoritative spec**: `WORKFLOW_JSON_SPEC.md`
- **Python validation contracts**: `state-schema.py`, `config-schema.py`
- **Example workflow**: `workflow.json` (customer support)
- **Migration guide**: `MIGRATION_GUIDE.md`
- **Skills architecture**: `llm-skills-architecture.md`
