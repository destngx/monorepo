# Workflow JSON Schema System

Local entrypoint for the prompt-driven workflow graph schema.

## Use this folder for

- workflow graph JSON entry docs
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
- **Extract per-node configuration** (provider, model, temperature, max_tokens, tools, custom prompts).
- Build LangGraph from the explicit node-and-edge graph with per-node metadata.

### 2. **Execution**

```
Entry Node → [Input Config] → Current State

For each node in the graph:
  1. Evaluate incoming edges
  2. Find first edge with true condition (or first unconditional edge)
  3. Execute the next node:
    - If agent_node:
      * Load per-node configuration (provider, model, temperature, max_tokens, tools, prompts)
      * Route to appropriate LLM provider (GitHub Copilot, OpenAI, etc.)
      * Load skills on demand (if in allowed tools list)
      * Execute with node-specific prompts and LLM settings
      * Store output in state
    - If branch: evaluate condition, don't execute work
    - If human_decision: pause, wait for user input
    - If exit: return final output
  4. Repeat until the exit node is reached (or limits exceeded)
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

### 5. **Per-Node LLM Configuration [MVP]**

Each agent_node can override global defaults:

```python
# Per-node config extracted at graph build time
node_config = {
    "provider": "github",                  # LLM provider (github for GitHub Copilot)
    "model": "claude-3.5-sonnet",          # Model name for provider
    "temperature": 0.7,                    # Sampling temperature [0, 1]
    "max_tokens": 2000,                    # Maximum output length
    "system_prompt": "You are...",         # Node-specific system prompt (overrides global)
    "user_prompt_template": "...",         # Node-specific user prompt (overrides global)
    "tools": ["load_skill", "search"]      # MCP tools this node can access (subset of global)
}

# At node execution time:
provider_client = get_provider_client(node_config["provider"], node_config["model"])
response = provider_client.call(
    system_prompt=node_config["system_prompt"],
    user_prompt=node_config["user_prompt_template"].format(**context),
    temperature=node_config["temperature"],
    max_tokens=node_config["max_tokens"],
    tools=node_config["tools"]  # Only these tools available to agent
)
```

---

## Common Patterns

### Pattern 1: Decision Graph

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
