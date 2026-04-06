# Workflow JSON Schema System

## Quick Start

A **workflow** is a declarative LangGraph definition in JSON that specifies:

1. **Nodes** — discrete work units (entry, skill execution, branching, exit)
2. **Edges** — transitions between nodes with conditions
3. **Limits** — execution constraints (timeouts, token budgets, hop limits)
4. **Metadata** — author, version, tags

**Skills** are separate: they're recipes/playbooks loaded at node execution time, not part of workflow structure.

### Minimal Example

```json
{
  "name": "classify-email",
  "version": "1.0.0",
  "nodes": [
    {
      "id": "start",
      "type": "entry",
      "config": {
        "properties": { "email_body": { "type": "string" } },
        "required": ["email_body"]
      }
    },
    {
      "id": "classify",
      "type": "skill_call",
      "skill_id": "classify_email",
      "config": {
        "input_mapping": { "text": "$.config.email_body" },
        "output_key": "classification"
      }
    },
    {
      "id": "end",
      "type": "exit",
      "config": {
        "output_mapping": { "category": "$.classification.output" }
      }
    }
  ],
  "edges": [
    { "from": "start", "to": "classify" },
    { "from": "classify", "to": "end" }
  ]
}
```

---

## Concepts

### Node Types

| Type               | Purpose                                   | Example                |
| ------------------ | ----------------------------------------- | ---------------------- |
| **entry**          | Workflow start; defines input schema      | Accept customer query  |
| **exit**           | Workflow end; maps outputs                | Return final decision  |
| **skill_call**     | Execute a skill with input/output mapping | Call "research" skill  |
| **branch**         | Evaluate condition (no work)              | If/then decision point |
| **human_decision** | Pause for human approval                  | Ask user to verify     |

### Edges

Connect nodes with optional conditions:

```json
{ "from": "classify", "to": "high_priority", "condition": "$.classification.priority == 'high'" },
{ "from": "classify", "to": "low_priority", "condition": "$.classification.priority == 'low'" }
```

Conditions are **JSONPath expressions** evaluated against workflow state.

### Skills

Skills are loaded **at node execution time**, not at graph build time.

When a `skill_call` node executes:

1. Runtime reads `skill_id` (e.g., "research").
2. Looks up skill in registry → finds `SKILL.md`.
3. Loads skill markdown (frontmatter + body + examples).
4. Injects into execution context.
5. Agent executes with skill guidance.
6. Output stored via `output_key`.

**Key insight**: Workflow structure is independent of skill availability.

### Input/Output Mapping

Use **JSONPath** to route data between nodes:

```json
{
  "id": "research_node",
  "config": {
    "input_mapping": {
      "topic": "$.config.topic", // From entry config
      "depth": "$.config.depth"
    },
    "output_key": "research_data" // Stored as $.research_data
  }
}
```

Access mapped data in subsequent nodes:

```json
{
  "from": "research_node",
  "to": "summarize",
  "condition": "$.research_data.confidence > 0.7"
}
```

### Guardrails

Protect sensitive node execution:

```json
{
  "id": "billing_agent",
  "type": "skill_call",
  "guardrails": {
    "input": {
      "max_tokens": 2000,
      "blocked_patterns": ["ignore instructions", "system prompt"]
    },
    "output": {
      "pii_detection": true,
      "required_format": "json",
      "blocked_topics": ["internal_pricing"]
    }
  }
}
```

Guardrails are evaluated by the runtime before/after node execution.

---

## File Structure

```
docs/graph-weave/
├── specification/
│   └── workflow-schema/
│       ├── WORKFLOW_JSON_SPEC.md          ← Full authoritative spec
│       └── MIGRATION_GUIDE.md             ← How to migrate from old model
├── code/
│   ├── workflow.json                      ← Example workflow (customer support)
│   └── workflow.schema.ts                 ← TypeScript validation schema
└── ...
```

---

## Validation

Use the TypeScript schema to validate workflows:

```typescript
import { validateWorkflow, validateWorkflowIntegrity } from './workflow.schema';

// Parse and validate
const result = validateWorkflow(jsonData);
if (!result.valid) {
  console.error('Validation failed:', result.errors);
}

// Check for structural issues (unreachable nodes, etc.)
const integrity = validateWorkflowIntegrity(result.workflow!);
if (!integrity.valid) {
  console.warn('Integrity issues:', integrity.issues);
}
```

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
     - If skill_call: load skill, execute, store output
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
    { "id": "classify", "type": "skill_call", "skill_id": "classify_request" },
    { "id": "route", "type": "branch", "config": { "condition_expression": "$.classification.type" } },
    { "id": "billing_agent", "type": "skill_call", "skill_id": "billing" },
    { "id": "shipping_agent", "type": "skill_call", "skill_id": "shipping" }
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
    { "id": "agent_a", "type": "skill_call", "skill_id": "analysis_a" },
    { "id": "agent_b", "type": "skill_call", "skill_id": "analysis_b" },
    { "id": "merge", "type": "skill_call", "skill_id": "aggregate" }
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

| Issue                      | Solution                                                      |
| -------------------------- | ------------------------------------------------------------- |
| **Unreachable node**       | Check incoming edges; all non-entry nodes need incoming path. |
| **Condition always false** | Verify JSONPath is correct; check state structure.            |
| **Missing skill_id**       | Add `"skill_id": "..."` to `skill_call` node config.          |
| **Circular loop**          | Ensure loop has explicit stagnation detection in limits.      |
| **Output mapping wrong**   | Verify JSONPath in exit node matches actual state keys.       |

---

## Migration from Old Model

See `MIGRATION_GUIDE.md` for detailed migration steps from subagent-routing workflows.

**Key changes**:

- ❌ Remove top-level `skills` array
- ❌ Remove `subagents` array
- ✅ Add explicit `nodes` array (entry, exit, skill_call, branch)
- ✅ Add explicit `edges` array with conditions
- ✅ Move skill refs to node-level `config.skill_id`

---

## References

- **Authoritative spec**: `WORKFLOW_JSON_SPEC.md`
- **TypeScript validation**: `workflow.schema.ts`
- **Example workflow**: `workflow.json` (customer support)
- **Migration guide**: `MIGRATION_GUIDE.md`
- **Skills architecture**: `llm-skills-architecture.md`
