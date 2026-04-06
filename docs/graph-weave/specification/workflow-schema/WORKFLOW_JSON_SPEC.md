# Workflow JSON Specification (Corrected LangGraph Model)

## 1. Objective

- **What**: Define the authoritative JSON schema for declaring LangGraph workflows where skills are loaded as node context (not routing logic).
- **Why**: Separate workflow structure (nodes, edges, conditions) from skill content (recipes, playbooks).
- **Who**: Workflow authors, runtime engineers, integrators.

## Traceability

- **FR-WF-001**: Workflow JSON must define explicit nodes and edges (LangGraph-native).
- **FR-WF-002**: Skills must be referenced in node definitions, not used to determine routing.
- **FR-WF-003**: Edge conditions must evaluate node results, not available skills.
- **FR-WF-004**: Guardrails must be attached to nodes, not the workflow level.
- **FR-WF-005**: Workflow structure must be serializable and execute on any LangGraph runtime.

## 2. Scope

- **In scope**: Node definition, edge transitions, edge conditions, input/output mapping, per-node guardrails.
- **Out of scope**: Skill content (skills are loaded separately at runtime); provider implementation details; multi-tenant tenant routing.

## 3. Key Concept: Separation of Concerns

```
┌─────────────────────────────────────┐
│       Workflow JSON (Structure)      │
├─────────────────────────────────────┤
│ • Nodes (discrete work units)        │
│ • Edges (transitions)                │
│ • Conditions (evaluated at runtime)  │
│ • Node refs to skills (not routing)  │
└─────────────────────────────────────┘
            ↓ (loads)
┌─────────────────────────────────────┐
│      Skills (Loaded Content)         │
├─────────────────────────────────────┤
│ • Recipes and procedures             │
│ • Examples and patterns              │
│ • Troubleshooting guides             │
│ • Context injected at node execution │
└─────────────────────────────────────┘
```

## 4. JSON Schema

### Top-Level Structure

```json
{
  "name": "string",
  "version": "semver",
  "description": "string",
  "metadata": {
    "author": "string",
    "tenant_id": "string (optional)",
    "created_at": "ISO8601",
    "tags": ["string"]
  },
  "limits": {
    "max_hops": "integer (default: 20)",
    "max_tokens": "integer (default: 100000)",
    "timeout_seconds": "integer (default: 300)",
    "stagnation_window": "integer (default: 3)",
    "stagnation_threshold": "integer (default: 2)"
  },
  "nodes": [
    { "node definition" }
  ],
  "edges": [
    { "edge definition" }
  ]
}
```

### Node Definition

Every node is a discrete work unit. The node type determines its behavior.

```json
{
  "id": "string (unique within workflow)",
  "type": "enum: entry | exit | skill_call | human_decision | branch",
  "display_name": "string (optional)",
  "description": "string (optional)",

  "config": {
    // Type-specific configuration
  },

  "guardrails": {
    "input": {
      "model": "string (e.g., llama-guard-3, optional)",
      "max_tokens": "integer (optional)",
      "blocked_patterns": ["string"]
    },
    "output": {
      "model": "string (optional)",
      "pii_detection": "boolean",
      "required_format": "string (e.g., markdown, json)",
      "blocked_topics": ["string"]
    }
  }
}
```

### Node Type: `entry`

Marks the start of the workflow. Accepts input config.

```json
{
  "id": "start",
  "type": "entry",
  "display_name": "Workflow Start",
  "config": {
    "properties": {
      "topic": { "type": "string", "description": "Research topic" },
      "depth": { "type": "integer", "description": "Research depth (1-5)" }
    },
    "required": ["topic"]
  }
}
```

### Node Type: `skill_call`

Executes a skill. The skill is loaded from the skill registry and injected as context.

```json
{
  "id": "research_node",
  "type": "skill_call",
  "display_name": "Research Phase",
  "description": "Conduct deep research using the research skill",

  "config": {
    "skill_id": "research",
    "input_mapping": {
      "topic": "$.config.topic",
      "depth": "$.config.depth"
    },
    "output_key": "research_result"
  },

  "guardrails": {
    "input": {
      "max_tokens": 2000,
      "blocked_patterns": ["ignore instructions"]
    },
    "output": {
      "pii_detection": true,
      "required_format": "markdown"
    }
  }
}
```

### Node Type: `branch`

Evaluates a condition and determines routing. Does not execute skill work itself.

```json
{
  "id": "quality_check",
  "type": "branch",
  "display_name": "Quality Gate",
  "description": "Assess if research is high-confidence",

  "config": {
    "condition_expression": "$.research_result.confidence > 0.7"
  }
}
```

### Node Type: `human_decision`

Awaits human input to proceed. Pauses execution.

```json
{
  "id": "approval",
  "type": "human_decision",
  "display_name": "Human Approval",
  "description": "Waiting for user to approve direction",

  "config": {
    "prompt": "Do you want to proceed with this plan?",
    "options": ["yes", "no", "revise"]
  }
}
```

### Node Type: `exit`

Marks the end of the workflow. Outputs final result.

```json
{
  "id": "end",
  "type": "exit",
  "display_name": "Workflow Complete",

  "config": {
    "output_mapping": {
      "final_report": "$.summary_result.output",
      "confidence": "$.research_result.confidence"
    }
  }
}
```

### Edge Definition

Edges define transitions between nodes. An edge evaluates a condition to decide if it should be taken.

```json
{
  "id": "string (optional, unique within workflow)",
  "from": "string (source node id)",
  "to": "string (target node id)",

  "condition": "JSONPath or boolean",

  "config": {
    "label": "string (optional, for visualization)"
  }
}
```

**Condition Types**:

1. **Implicit True** (always taken):

   ```json
   { "from": "start", "to": "research_node" }
   ```

2. **JSONPath Evaluation**:

   ```json
   {
     "from": "research_node",
     "to": "summarize",
     "condition": "$.research_result.confidence > 0.7"
   }
   ```

3. **Negation**:
   ```json
   {
     "from": "research_node",
     "to": "rework",
     "condition": "!($.research_result.confidence > 0.7)"
   }
   ```

## 5. Complete Example: Research Workflow

```json
{
  "name": "research-workflow",
  "version": "1.0.0",
  "description": "Multi-stage research with confidence gates",

  "metadata": {
    "author": "ai-researcher",
    "created_at": "2026-04-07T00:00:00Z",
    "tags": ["research", "analysis"]
  },

  "limits": {
    "max_hops": 10,
    "max_tokens": 150000,
    "timeout_seconds": 600
  },

  "nodes": [
    {
      "id": "start",
      "type": "entry",
      "display_name": "Input",
      "config": {
        "properties": {
          "topic": { "type": "string" },
          "depth": { "type": "integer", "minimum": 1, "maximum": 5 }
        },
        "required": ["topic"]
      }
    },
    {
      "id": "research_phase",
      "type": "skill_call",
      "display_name": "Research",
      "description": "Conduct initial research using the research skill",

      "config": {
        "skill_id": "research",
        "input_mapping": {
          "topic": "$.config.topic",
          "depth": "$.config.depth"
        },
        "output_key": "research_data"
      },

      "guardrails": {
        "input": {
          "max_tokens": 3000,
          "blocked_patterns": ["ignore previous"]
        },
        "output": {
          "pii_detection": true,
          "required_format": "markdown"
        }
      }
    },
    {
      "id": "confidence_gate",
      "type": "branch",
      "display_name": "Confidence Check",
      "description": "Evaluate research quality",
      "config": {
        "condition_expression": "$.research_data.confidence > 0.75"
      }
    },
    {
      "id": "summarize_phase",
      "type": "skill_call",
      "display_name": "Summarization",
      "description": "Create executive summary",

      "config": {
        "skill_id": "summarize",
        "input_mapping": {
          "research_text": "$.research_data.output",
          "max_length": 500
        },
        "output_key": "summary"
      }
    },
    {
      "id": "rework_phase",
      "type": "skill_call",
      "display_name": "Rework Research",
      "description": "Deepen research due to low confidence",

      "config": {
        "skill_id": "research",
        "input_mapping": {
          "topic": "$.config.topic",
          "depth": "5"
        },
        "output_key": "research_data"
      }
    },
    {
      "id": "end",
      "type": "exit",
      "display_name": "Complete",
      "config": {
        "output_mapping": {
          "summary": "$.summary.output",
          "confidence": "$.research_data.confidence",
          "metadata": {
            "workflow": "research-workflow",
            "completed_at": "NOW()"
          }
        }
      }
    }
  ],

  "edges": [
    {
      "from": "start",
      "to": "research_phase",
      "config": { "label": "Initialize research" }
    },
    {
      "from": "research_phase",
      "to": "confidence_gate",
      "config": { "label": "Assess quality" }
    },
    {
      "from": "confidence_gate",
      "to": "summarize_phase",
      "condition": "$.research_data.confidence > 0.75",
      "config": { "label": "High confidence → summarize" }
    },
    {
      "from": "confidence_gate",
      "to": "rework_phase",
      "condition": "!($.research_data.confidence > 0.75)",
      "config": { "label": "Low confidence → rework" }
    },
    {
      "from": "summarize_phase",
      "to": "end",
      "config": { "label": "Output result" }
    },
    {
      "from": "rework_phase",
      "to": "confidence_gate",
      "config": { "label": "Re-evaluate" }
    }
  ]
}
```

## 6. How Skills Load at Runtime

1. **Workflow JSON Parsed**: Runtime reads nodes and edges.
2. **Node Identified**: Runtime encounters `"skill_id": "research"`.
3. **Skill Loaded**: Runtime fetches skill from registry (SKILL.md frontmatter + body).
4. **Context Injected**: Skill markdown (with examples, procedures) injected into node execution context.
5. **Execution**: Agent executes with skill guidance + node input mapping.
6. **Output Mapped**: Node output stored via `output_key`.
7. **Edge Evaluated**: Next node determined by edge conditions (not skill availability).

**Key Point**: The workflow structure doesn't change based on available skills. Skills are **context**, not **routing logic**.

## 7. Verification Checklist

- [ ] All nodes have unique IDs within the workflow.
- [ ] Every `skill_call` node references a valid `skill_id` (or is gracefully skipped).
- [ ] All edges reference existing source and target nodes.
- [ ] Conditions use valid JSONPath syntax and reference accessible state paths.
- [ ] Entry and exit nodes are present and correctly linked.
- [ ] No circular loops without explicit stagnation handling.
- [ ] Input/output mappings are valid JSONPath expressions.
- [ ] Guardrails are optional but, if present, have valid models or patterns.

## 8. Runtime Contract

**Input**:

- Workflow JSON (above schema)
- Initial config (matches entry node properties)
- Loaded skills (skill_id → SKILL.md mapping)

**Output**:

- Final state dict (structured via exit node config)
- Execution trace (node visits, condition evaluations, timings)

**Guarantees**:

- Nodes execute in order defined by edges and conditions.
- State is immutable during edge evaluation (snapshots for condition checking).
- Skill loading is non-blocking (skills load in background if needed).
- Execution halts on max_hops, timeout, or explicit exit.

---

## Differences from Previous Approaches

| Aspect         | Hybrid Config (Rejected)          | Explicit State Machine (Adopted)                           |
| -------------- | --------------------------------- | ---------------------------------------------------------- |
| **Routing**    | Determined by available skills    | Determined by edge conditions on state                     |
| **Node Type**  | Implicit (subagent + skill_id)    | Explicit (entry, exit, skill_call, branch, human_decision) |
| **Skill Role** | Constrains workflow DAG           | Loaded as node execution context                           |
| **Structure**  | Workflow adapts to skills         | Skills adapt to workflow structure                         |
| **Modularity** | Skills + workflow tightly coupled | Clear separation of structure and content                  |

---

## Next Steps

1. **Implement Workflow Parser**: Load JSON, validate schema, build LangGraph.
2. **Implement Skill Loader**: Resolve skill_id → load SKILL.md frontmatter + body.
3. **Implement Executor**: Execute nodes, evaluate edge conditions, handle guardrails.
4. **Build Registry**: Map skill_id to SKILL.md location (file-based or HTTP).
5. **Test Suite**: Validate schema compliance and execution correctness.
