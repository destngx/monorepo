# VERIFY-GW-WF-002: Prompt-driven workflow patterns and LLM agent mode

> **Linked Task** : GW-WF-002 — `docs/graph-weave/specification/workflow-schema/tasks/SPEC/GW-WF-002.md`
> **Verification Types** : SCHEMA, FUNC, INTG
> **Phase ID** : SPEC
> **Risk Level** : High
> **Reviewer** : TBD
> **Verified On** : 2026-04-08
> **Overall Status** : Pass

---

## 1. Traceability

The specification must reference:

- `docs/graph-weave/specification/workflow-schema/plan/prompt-driven-agent-model.md`
- `docs/graph-weave/specification/workflow-schema/README.md`
- `docs/graph-weave/specification/runtime/plan/universal-interpreter-and-skill-loading.md`
- GW-WF-001 (workflow schema locked)
- GW-SKILL-001, GW-SKILL-002 (skill discovery and loading)

**Evidence**: Cross-references confirm alignment with schema and skills specifications.

---

## 2. Scope Compliance

Verify task produced one deliverable: prompt-driven workflow model specification with LLM agent contract.

| Criterion                     | Expected                               | Status |
| ----------------------------- | -------------------------------------- | ------ |
| Single deliverable            | One prompt-driven model specification  | pass   |
| No model selection/tuning     | Defers to MVP phase                    | pass   |
| No prompt engineering details | Defers to implementation               | pass   |
| No token counting/cost estim. | Defers to runtime monitoring           | pass   |
| No multi-turn conversation    | Agent internal; defers to agent design | pass   |

---

## 3. Type-Specific Criteria

### 3.1 SCHEMA (LLM Agent Contract)

| #         | Criterion                          | Expected                                   | Actual | Status |
| --------- | ---------------------------------- | ------------------------------------------ | ------ | ------ |
| SCHEMA-01 | Agent mode defined                 | Prompt-driven agent mode is defined        | pass   | pass   |
| SCHEMA-02 | System prompt structure documented | Goal, available skills, state, next step   | pass   | pass   |
| SCHEMA-03 | System prompt fields specified     | Required sections and content              | pass   | pass   |
| SCHEMA-04 | Skill list injection documented    | How skills are provided to LLM             | pass   | pass   |
| SCHEMA-05 | Skill metadata format documented   | What metadata is injected                  | pass   | pass   |
| SCHEMA-06 | LLM output contract specified      | JSON/structured format                     | pass   | pass   |
| SCHEMA-07 | Output fields documented           | reasoning, action, skill_name, skill_input | pass   | pass   |
| SCHEMA-08 | Examples provided                  | Sample system prompts and LLM responses    | pass   | pass   |

### 3.2 FUNC (Agent Behavior)

| #       | Criterion                      | Expected                              | Actual | Status |
| ------- | ------------------------------ | ------------------------------------- | ------ | ------ |
| FUNC-01 | Agent receives execution state | How state is passed to LLM            | pass   | pass   |
| FUNC-02 | Agent generates valid actions  | Next node or skill call only          | pass   | pass   |
| FUNC-03 | Output validation documented   | How system validates LLM response     | pass   | pass   |
| FUNC-04 | Fallback behavior specified    | What happens if LLM output is invalid | pass   | pass   |

### 3.3 INTG (Skill & Workflow Integration)

| #       | Criterion                         | Expected                            | Actual | Status |
| ------- | --------------------------------- | ----------------------------------- | ------ | ------ |
| INTG-01 | Skill discovery integration       | Agent can see available skills      | pass   | pass   |
| INTG-02 | Execution history passed to agent | Messages/context from prior nodes   | pass   | pass   |
| INTG-03 | Workflow schema compatibility     | Agent respects node types and edges | pass   | pass   |

**Supporting Artifacts**:

- System prompt template
- Skill metadata injection format
- LLM output validation schema
- Fallback decision tree
- Example workflow: decision graph, quality gate, parallel processing
- State/history structure

**Notes**:

> [To be filled after task completion]

---

## 4. Documentation Check

Required updates:

- [x] `docs/graph-weave/specification/workflow-schema/plan/prompt-driven-agent-model.md` — decisions confirmed
- [x] `docs/graph-weave/specification/workflow-schema/README.md` — LLM agent mode documented
- [x] Reference to skill metadata (from GW-SKILL-001) available
- [x] Reference to skill loading (from GW-SKILL-002) available

---

## 5. Final Decision

| Decision            | Condition                                                       |
| ------------------- | --------------------------------------------------------------- |
| **Pass**            | Agent contract + prompt structure + output format all defined   |
| **Needs Revision**  | Missing contract field or validation rules unclear; agent fixes |
| **Fail + Rollback** | Agent mode undefined or incompatible with skills; rejected      |

**Decision**: Pass

**Rationale**:

> [To be filled by reviewer after task completion]

**Reviewer Signature**: `[agent-name]` — `[YYYY-MM-DD]`
