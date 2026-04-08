# GW-WF-002: Specify prompt-driven workflow patterns and LLM agent mode

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : LLM-driven workflow execution
- **Feature** : Define how LLM agents generate workflow steps at runtime
- **Rationale** : Prompt-driven mode is key differentiator; must be locked before MOCK to prevent mid-implementation strategy changes

---

### Input

- **Data / Files** : `[[../specification/workflow-schema/plan/prompt-driven-agent-model.md]]`, `[[../specification/workflow-schema/README.md]]`, `[[../specification/runtime/plan/universal-interpreter-and-skill-loading.md]]`
- **Dependencies** : GW-WF-001 (workflow schema must be defined), GW-SKILL-001 (skills must be discoverable)
- **External Systems**: LangGraph, LLM provider (e.g., Claude, GPT-4)

---

### Scope

- **In Scope** :
  - Define prompt-driven mode (vs. graph-driven mode)
  - Document LLM agent contract (system prompt, available skills, expected output)
  - Define how LLM generates next node/action from state
  - Document skill discovery in prompt context
  - Define how LLM is provided with execution state and history
  - Define output parsing and validation (must be valid node or skill call)

- **Out of Scope**:
  - LLM model selection or fine-tuning
  - Prompt engineering details
  - Token counting or cost estimation
  - Multi-turn conversation management (agent internal)

- **Max Increment**: Complete prompt-driven model specification

---

### Approach

1. Synthesize prompt-driven-agent-model and plan docs
2. Define agent system prompt structure and content
3. Document skill list injection (how skills are provided to LLM)
4. Define agent output format (expected JSON or structured output)
5. Document validation and fallback if LLM output is invalid

**Files to Modify/Create**:

- `docs/graph-weave/specification/workflow-schema/plan/prompt-driven-agent-model.md` — Confirm agent model design

---

### Expected Output

- **Deliverable** : Prompt-driven workflow model specification
- **Format** : Markdown with prompt templates and JSON contract examples
- **Example** :

```
System Prompt Structure:
1. Goal: Execute workflow "{workflow_name}"
2. Available Skills: [JSON array of skill metadata]
3. Current State: [execution state, messages, last node]
4. Next Step: Generate next action or call a skill

LLM Output Contract (JSON):
{
  "reasoning": "...",
  "action": "call_skill" | "goto_node" | "done",
  "skill_name": "...",
  "skill_input": {...}
}
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-WF-002.md]]`

---

### References

- `[[../specification/workflow-schema/plan/prompt-driven-agent-model.md]]` — Agent model decisions; must be reflected
- `[[./GW-SKILL-001.md]]` — Skills must be discoverable for injection into prompt
- `[[./GW-SKILL-002.md]]` — Skill loading; agent needs access to skill metadata
- `[[./GW-RUNTIME-001.md]]` — Interpreter interface; agent operates inside it
