# Workflow JSON Specification (Prompt-Driven Agent Model)

## 1. Objective

- **What**: Define the authoritative JSON schema for declaring LangGraph workflows where agents execute autonomously based on system + user prompts and load skills dynamically at runtime.
- **Why**: Treat nodes and edges as the core workflow graph, separate that graph from skill content (recipes), and let agent behavior emerge from node prompts. Allow agents to decide which skills to load based on task context.
- Workflow metadata, nodes, edges, limits, and guardrails must be explicit and serializable.
- **Who**: Workflow authors, runtime engineers, integrators.

## Traceability

- **FR-WF-001 [MOCK,MVP,FULL]**: Workflow JSON must define explicit nodes and edges as the source of truth for workflow structure.
- **FR-WF-002 [MOCK,MVP,FULL]**: Nodes must specify system and user prompts; agents decide skill loading autonomously.
- **FR-WF-003 [MOCK,MVP,FULL]**: Edge conditions must evaluate node results (JSONPath), not available skills or agent choices.
- **FR-WF-004 [MOCK,MVP,FULL]**: Guardrails must be attached to nodes for input/output validation.
- **FR-WF-005 [MVP,FULL]**: Workflow structure must be serializable and execute on any LangGraph runtime with MCP tool support.

## 2. Scope

- **In scope**: Node definition (agent_node types with prompts), edge transitions, edge conditions, input/output mapping, per-node guardrails.
- **Out of scope**: Skill content (loaded dynamically at agent runtime); LLM model selection; MCP tool implementations.

## 3. Key Concept: Prompt-Driven Agent Autonomy

```
┌─────────────────────────────────────┐
│       Workflow JSON (Structure)      │
├─────────────────────────────────────┤
│ • Nodes (agent_node with prompts)    │
│ • Edges (transitions)                │
│ • Conditions (evaluated on output)   │
│ • Limits (safety bounds)             │
│ • NO explicit skill references       │
└─────────────────────────────────────┘
            ↓ (manages)
┌─────────────────────────────────────┐
│      Agent Execution (Per Node)      │
├─────────────────────────────────────┤
│ • System prompt: Role definition     │
│ • User prompt: Task description      │
│ • Agent reads prompt context         │
│ • Agent calls load_skill() as needed │
│ • Agent executes autonomously        │
│ • Output stored in state             │
└─────────────────────────────────────┘
            ↓ (feeds output to)
┌─────────────────────────────────────┐
│      Edge Routing (Deterministic)    │
├─────────────────────────────────────┤
│ • Condition: JSONPath expression     │
│ • Evaluate on agent output           │
│ • First true condition → next node   │
└─────────────────────────────────────┘
            ↓ (loads if needed)
┌─────────────────────────────────────┐
│    Skills (External Content)         │
├─────────────────────────────────────┤
│ • Markdown with YAML frontmatter     │
│ • Loaded dynamically by agent        │
│ • Independent of workflow            │
│ • Can update without redeployment    │
└─────────────────────────────────────┘
```

**Key Principle**: Nodes and edges define the workflow graph. Agents decide which skills to load via `load_skill()` MCP tool based on task context. Workflow doesn't know about specific skills—it only defines structure and edge routing.

- Nodes must define `id`, `type`, `config`, and guardrails; edges must define `from`, `to`, and `condition`.
- Schema versioning must preserve forward and backward compatibility where possible.

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

Every node is a discrete work unit inside the workflow graph. The node type determines its behavior.

```json
{
  "id": "string (unique within workflow)",
  "type": "enum: entry | exit | agent_node | human_decision | branch",
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

### Node Type: `agent_node`

Executes an LLM agent with system and user prompts. The agent autonomously decides which skills to load via `load_skill()` MCP tool.

```json
{
  "id": "research_node",
  "type": "agent_node",
  "display_name": "Research Phase",
  "description": "Conduct research using agent-selected skills",

  "config": {
    "system_prompt": "You are a thorough researcher. You have access to:\n- load_skill(skill_name): Load a research playbook\n- search(query): Real-time web search\n- verify(claim): Fact-check information\n\nBefore starting, load the research skill to guide your work. Always cite sources and provide confidence scores based on evidence quality.",

    "user_prompt_template": "Research the following topic:\n{topic}\n\nProvide:\n1. Key findings with sources\n2. Confidence score (0-1)\n3. Known limitations",

    "input_mapping": {
      "topic": "$.config.topic",
      "depth": "$.config.depth"
    },

    "output_key": "research_output",

    "output_schema": {
      "type": "object",
      "properties": {
        "findings": { "type": "string" },
        "sources": { "type": "array", "items": { "type": "string" } },
        "confidence": { "type": "number", "minimum": 0, "maximum": 1 }
      },
      "required": ["findings", "confidence"]
    }
  },

  "guardrails": {
    "input": {
      "max_tokens": 5000,
      "blocked_patterns": ["ignore instructions", "system prompt"]
    },
    "output": {
      "pii_detection": true,
      "schema_validation": true,
      "required_format": "json"
    }
  },

  "retry_config": {
    "max_attempts": 2,
    "on_error": "retry",
    "timeout_seconds": 120
  }
}
```

**System Prompt Best Practices**:

1. **Role definition**: "You are a [role] with expertise in [domain]"
2. **Tool availability**: List all available tools: `load_skill()`, `search()`, `verify()`, `analyze()`
3. **Skill loading instruction**: "Load the [skill_name] skill to guide your work" or "Select appropriate skills based on task"
4. **Output format**: "Return JSON with fields: {field1, field2, ...}"
5. **Constraints**: "Do not... You must... Always..."
6. **Quality standards**: "Be thorough, verify claims, cite sources, report confidence"

**User Prompt Best Practices**:

1. **Use templates**: `{topic}`, `{context}`, `{user_input}` (interpolated from input_mapping)
2. **Clear instructions**: Specify what agent should do and what format to return
3. **Context inclusion**: Include relevant details that agent needs to succeed
4. **Output schema**: Tell agent what fields to include in response

### Node Type: `branch`

Evaluates a condition and determines routing. Does not execute skill work itself.

```json
{
  "id": "quality_check",
  "type": "branch",
  "display_name": "Quality Gate",
  "description": "Assess if research is high-confidence",

  "config": {
    "condition_expression": "$.research_output.confidence > 0.7"
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
      "final_report": "$.summary_output.text",
      "confidence": "$.research_output.confidence"
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
     "condition": "$.research_output.confidence > 0.7"
   }
   ```

3. **Negation**:
   ```json
   {
     "from": "research_node",
     "to": "rework",
     "condition": "!($.research_output.confidence > 0.7)"
   }
   ```

### 4.1 Edge Condition Evaluation Rules [MVP]

**JSONPath Expression Evaluation**:

- Conditions are evaluated as JSONPath expressions against the current workflow state snapshot.
- The runtime must snapshot the state before evaluating all outgoing edges from a node.
- State is immutable during edge evaluation (prevents race conditions and ensures determinism).
- If a JSONPath path does not exist in state, evaluation must treat it as `null`.
- Operators: `>`, `<`, `>=`, `<=`, `==`, `!=`, `!` (negation)
- Nested path access: `$.field.nested.path` follows JSONPath dot notation.

**None/Null Handling**:

- If a referenced path is `null`, comparison with literals follows JSON semantics:
  - `null > 0` evaluates to `false`
  - `null == null` evaluates to `true`
  - `null != "value"` evaluates to `true`
- Agents MUST ensure output keys exist in state before edge evaluation (guardrails validate this).
- If a required output key is missing, treat as validation error and force exit.

**Determinism Guarantee**:

- Given the same node output and state snapshot, the same edge is always taken (JSONPath evaluation is deterministic).
- This enables replay, testing, and audit trails.

**Edge Routing Order**:

- The runtime evaluates edges in the order they are defined in the workflow JSON.
- The first edge with a `true` condition is taken.
- Unconditional edges (no condition field) are always `true`.
- If no edge condition is true, the workflow enters a `stagnation_detected` state and is forced to exit.

### 4.2 State Propagation and Mutation Rules [MVP]

**State Structure**:

- Workflow state is a flat-or-nested JSON object that accumulates outputs from each node.
- Node outputs are stored at keys defined in the node's `output_key` field.
- State persists across all nodes in the workflow.
- State is NOT cleared between nodes; outputs are additive.

**Output Storage**:

```python
# Example state accumulation:
state = {
  "entry": { "topic": "AI ethics", "depth": "thorough" },
  "research_output": { "findings": "...", "confidence": 0.85 },
  "verification_output": { "verified_facts": [...], "confidence": 0.9 },
  "summary_output": { "text": "...", "key_points": [...] }
}
```

**State Immutability During Edge Evaluation**:

```python
# Correct: Snapshot before evaluating edges
state_snapshot = copy.deepcopy(workflow_state)
eligible_edges = [
    e for e in outgoing_edges
    if evaluate_condition(e.condition, state_snapshot)
]
next_node = eligible_edges[0].to

# Incorrect: Mutating state while evaluating edges
# This breaks determinism and causes race conditions
```

**Input Mapping**:

- Node `input_mapping` fields use JSONPath to extract values from state for template substitution.
- Template placeholders (e.g., `{topic}`, `{research}`) are replaced with values from state at node execution time.
- If a mapped path doesn't exist, the runtime must raise a validation error and exit.

## 5. Complete Example: Multi-Agent Research Workflow

```json
{
  "name": "multi-agent-research",
  "version": "1.0.0",
  "description": "Research with agent autonomy and confidence-based routing",

  "metadata": {
    "author": "ai-researcher",
    "created_at": "2026-04-07T00:00:00Z",
    "tags": ["research", "analysis", "multi-agent"]
  },

  "limits": {
    "max_hops": 10,
    "max_tokens": 250000,
    "timeout_seconds": 600
  },

  "nodes": [
    {
      "id": "entry",
      "type": "entry",
      "display_name": "Input Configuration",
      "config": {
        "properties": {
          "topic": { "type": "string", "description": "Topic to research" },
          "depth": { "type": "string", "enum": ["quick", "thorough"] }
        },
        "required": ["topic"]
      }
    },

    {
      "id": "research",
      "type": "agent_node",
      "display_name": "Research Agent",
      "description": "Conduct research using agent-selected skills",

      "config": {
        "system_prompt": "You are a research analyst with deep subject matter expertise. You have access to:\n- load_skill(skill_name): Load skill guides (e.g., 'research', 'investigation', 'analysis')\n- search(query): Real-time web search\n- verify(claim): Fact-check assertions\n\nBefore starting, load the 'research' skill to guide your methodology. Always cite sources. Report confidence based on source quality and agreement between sources.",

        "user_prompt_template": "Research the topic: {topic}\n\nDepth requested: {depth}\n\nProvide:\n1. Key findings with sources\n2. Confidence score (0-1) \n3. Known gaps or uncertainties\n4. Recommended next steps if confidence is low",

        "input_mapping": {
          "topic": "$.entry.topic",
          "depth": "$.entry.depth"
        },

        "output_key": "research_output",

        "output_schema": {
          "type": "object",
          "properties": {
            "findings": { "type": "string" },
            "sources": { "type": "array", "items": { "type": "string" } },
            "confidence": { "type": "number", "minimum": 0, "maximum": 1 },
            "gaps": { "type": "string" }
          },
          "required": ["findings", "confidence"]
        }
      },

      "guardrails": {
        "input": {
          "max_tokens": 3000,
          "blocked_patterns": ["ignore instructions", "system prompt"]
        },
        "output": {
          "pii_detection": true,
          "schema_validation": true,
          "required_format": "json"
        }
      },

      "retry_config": {
        "max_attempts": 2,
        "on_error": "retry",
        "timeout_seconds": 180
      }
    },

    {
      "id": "confidence_check",
      "type": "branch",
      "display_name": "Confidence Gate",
      "description": "Route based on research confidence",
      "config": {
        "condition_expression": "$.research_output.confidence"
      }
    },

    {
      "id": "verify",
      "type": "agent_node",
      "display_name": "Verification Agent",
      "description": "Verify research findings for accuracy",

      "config": {
        "system_prompt": "You are a fact-checker and verification specialist. You have access to:\n- load_skill(skill_name): Load verification skill\n- verify(claim): Fact-check assertions\n- search(query): Counter-evidence search\n\nLoad the 'verification' skill. Systematically verify each key finding. Report verified, disputed, and unverifiable claims.",

        "user_prompt_template": "Verify these research findings:\n{findings}\n\nReport:\n1. Verified facts\n2. Disputed or uncertain claims\n3. Missing evidence\n4. Overall verification confidence (0-1)",

        "input_mapping": {
          "findings": "$.research_output.findings"
        },

        "output_key": "verification_output"
      },

      "guardrails": {
        "output": {
          "pii_detection": true,
          "schema_validation": true
        }
      }
    },

    {
      "id": "summarize",
      "type": "agent_node",
      "display_name": "Summarization Agent",
      "description": "Create executive summary",

      "config": {
        "system_prompt": "You are a technical writer. Load the 'summarization' skill. Create clear, concise summaries that highlight key points and integrate verification results.",

        "user_prompt_template": "Summarize these results:\nResearch: {research}\nVerification: {verification}\n\nCreate a concise summary with key findings, verified status, and confidence rating.",

        "input_mapping": {
          "research": "$.research_output.findings",
          "verification": "$.verification_output"
        },

        "output_key": "summary_output"
      }
    },

    {
      "id": "exit",
      "type": "exit",
      "display_name": "Output",
      "config": {
        "output_mapping": {
          "research": "$.research_output",
          "verification": "$.verification_output",
          "summary": "$.summary_output"
        }
      }
    }
  ],

  "edges": [
    {
      "from": "entry",
      "to": "research",
      "config": { "label": "Start research" }
    },
    {
      "from": "research",
      "to": "confidence_check",
      "config": { "label": "Evaluate confidence" }
    },
    {
      "from": "confidence_check",
      "to": "verify",
      "condition": "$.research_output.confidence > 0.6",
      "config": { "label": "Confidence adequate → verify" }
    },
    {
      "from": "confidence_check",
      "to": "research",
      "condition": "$.research_output.confidence <= 0.6",
      "config": { "label": "Low confidence → retry research" }
    },
    {
      "from": "verify",
      "to": "summarize",
      "config": { "label": "Proceed to summary" }
    },
    {
      "from": "summarize",
      "to": "exit",
      "config": { "label": "Complete" }
    }
  ]
}
```

## 6. How Agent Autonomy Works at Runtime

### Execution Flow

```
1. NODE EXECUTION (agent_node)
   ├─ Inputs: {topic: "AI ethics", depth: "thorough"}
   ├─ System Prompt: "You are a researcher. Load skills as needed."
   ├─ User Prompt: "Research: AI ethics, depth: thorough"

2. AGENT READS PROMPTS (LLM + MCP Tools)
   ├─ LLM analyzes: "I need to research, so I should load research skill"
   ├─ LLM calls MCP tool: load_skill("research")
   ├─ MCP returns: Skill definition (Markdown with YAML frontmatter)
   ├─ LLM reads skill: "Research methodology steps..."
   ├─ LLM may also call: search(), verify(), analyze() as needed
   ├─ LLM outputs: {findings: "...", confidence: 0.85, sources: [...]}

3. OUTPUT STORED
   ├─ Node stores output at key: "research_output"
   ├─ State updated: {research_output: {findings: ..., confidence: 0.85}}

4. EDGE EVALUATION (Workflow)
   ├─ Condition: "$.research_output.confidence > 0.6"
   ├─ Evaluation: 0.85 > 0.6? YES
   ├─ Next node: verify (if exists) or summarize

5. GUARDRAILS
   ├─ Pre-execution: Validate inputs (token count, blocked patterns)
   ├─ Post-execution: Validate outputs (schema, PII, format)
```

### Key Differences from Explicit Skill References

| Aspect                  | Old (skill_call)                       | New (agent_node + prompt-driven)                    |
| ----------------------- | -------------------------------------- | --------------------------------------------------- |
| **Skill selection**     | Explicit in workflow `skill_id`        | Agent decides via `load_skill()`                    |
| **Skill loading**       | Runtime loads specified skill          | Agent requests skill on demand                      |
| **Agent autonomy**      | Limited (follows fixed skill)          | High (can load multiple skills, use other tools)    |
| **Workflow coupling**   | Tight (workflow knows all skills)      | Loose (workflow only defines structure)             |
| **Flexibility**         | Low (skill changes = workflow changes) | High (skill updates don't require workflow changes) |
| **Multi-skill support** | No (one skill per node)                | Yes (agent can load multiple skills)                |

**Bottom Line**: The graph defines WHAT nodes do (research, verify, summarize) and HOW to route between them (based on confidence). Agents decide WHICH skills to load based on task context.

## 7. Verification Checklist

- [ ] All nodes have unique IDs within the workflow graph.
- [ ] Every `agent_node` has both `system_prompt` and `user_prompt_template` defined.
- [ ] All `user_prompt_template` placeholders (e.g., `{topic}`) have corresponding keys in `input_mapping`.
- [ ] All edges reference existing source and target nodes.
- [ ] Conditions use valid JSONPath syntax and reference accessible state paths.
- [ ] Edge conditions handle `null` values correctly (null comparisons must be intentional).
- [ ] State propagation is additive: each node output adds a key to the accumulated state.
- [ ] No missing required output keys: all paths referenced in edge conditions must exist in node outputs.
- [ ] Every node definition includes required metadata and guardrails where applicable.
- [ ] Entry and exit nodes are present and correctly linked.
- [ ] No circular loops without explicit stagnation handling in limits.
- [ ] Input/output mappings are valid JSONPath expressions.
- [ ] Output schemas in `agent_node` are valid JSON Schema.
- [ ] Guardrails (if present) have valid models or patterns.
- [ ] Edge routing order is deterministic (first true condition taken).

## 8. Runtime Contract

**Input**:

- Workflow JSON (above schema)
- Initial config (matches entry node properties)
- Available MCP tools for agents (load_skill, search, verify, analyze)

**Output**:

- Final state dict (structured via exit node config)
- Execution trace (node visits, condition evaluations, timings)

**Guarantees**:

- Nodes execute in order defined by edges and conditions.
- State is immutable during edge evaluation (snapshots for condition checking).
- Agent skill loading is non-blocking (agents call load_skill() synchronously, tool returns skill definition).
- Execution halts on max_hops, timeout, or explicit exit.
- **Determinism at edge level**: Given same node output, same edge is taken (JSONPath evaluation is deterministic).
- **Agent autonomy within guardrails**: Agent can choose skills/tools within constraints defined by prompts and guardrails.

---

## 9. Differences from Previous Approaches

| Aspect                    | Hybrid Config (Rejected)          | Explicit State Machine + skill_call (Old)  | Prompt-Driven agent_node (Current)                       |
| ------------------------- | --------------------------------- | ------------------------------------------ | -------------------------------------------------------- |
| **Routing**               | Determined by available skills    | Determined by `skill_id` + edge conditions | Determined by edge conditions on agent output            |
| **Skill Loading**         | Implicit (workflow decides)       | Explicit (workflow specifies `skill_id`)   | Dynamic (agent requests via `load_skill()`)              |
| **Node Type**             | Implicit (subagent + skill_id)    | Explicit (skill_call)                      | Explicit (agent_node with prompts)                       |
| **Agent Role**            | Fixed (execute skill)             | Fixed (execute skill)                      | Autonomous (decide which skills to load)                 |
| **Skill Ref in Workflow** | Yes                               | Yes (explicit `skill_id`)                  | No (implicit via prompts)                                |
| **Multi-Skill Support**   | No                                | No (one skill per node)                    | Yes (agent can load multiple skills)                     |
| **Modularity**            | Skills + workflow tightly coupled | Skills + workflow coupled via `skill_id`   | Clear separation: workflow = structure, skills = content |

---

## 10. Next Steps

1. **Create SYSTEM_PROMPT_GUIDELINES.md**: Best practices for writing effective system prompts for agent_node types.
2. **Design load_skill() MCP Tool**: Specification for how agents load skills at runtime.
3. **Implement Workflow Parser**: Load JSON, validate schema, build LangGraph with agent_node types.
4. **Implement Skill Discovery**: How agents find and enumerate available skills at runtime.
5. **Implement Agent Executor**: Execute LLM agents with system + user prompts, handle MCP tool calls.
6. **Build Example Workflows**: Convert existing workflows to prompt-driven agent_node format.
7. **Test Suite**: Validate schema compliance, agent autonomy, edge routing determinism, and guardrails.

---

## 11. Key Takeaways

1. **Workflow = Structure**: Nodes, edges, conditions. No explicit skill references.
2. **Prompts = Behavior**: System prompts define agent role; user prompts define task.
3. **Agent Autonomy = Flexibility**: Agents load skills dynamically based on task context.
4. **Edge Conditions = Determinism**: Routing is deterministic, based on agent output and JSONPath evaluation.
5. **No Coupling**: Workflow can remain unchanged when skills are added/updated.
