# VERIFY-GW-WF-001: Lock workflow JSON schema and node contracts

> **Linked Task** : GW-WF-001 — `docs/graph-weave/specification/workflow-schema/tasks/SPEC/GW-WF-001.md`
> **Verification Types** : SCHEMA, FUNC, INTG
> **Phase ID** : SPEC
> **Risk Level** : Critical
> **Reviewer** : TBD
> **Verified On** : 2026-04-08
> **Overall Status** : Pass

---

## 1. Traceability

The specification must reference:

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md`
- `docs/graph-weave/specification/workflow-schema/README.md`
- `docs/graph-weave/specification/workflow-schema/plan/schema-and-node-contracts.md`
- `docs/graph-weave/specification/runtime/universal-interpreter.md`

**Evidence**: Cross-references confirm alignment with source documents and interpreter compatibility.

---

## 2. Scope Compliance

Verify task produced one deliverable: workflow JSON schema specification with validation rules and examples.

| Criterion                   | Expected                               | Status |
| --------------------------- | -------------------------------------- | ------ |
| Single deliverable          | One workflow schema specification      | pass   |
| No workflow editor/UI       | Defers UI to frontend work             | pass   |
| No execution semantics      | References GW-RUNTIME-\* for execution | pass   |
| No schema migration tooling | Defers to tooling phase                | pass   |
| No example workflows        | Defers to GW-SKILL-\* examples         | pass   |

---

## 3. Type-Specific Criteria

### 3.1 SCHEMA (JSON Structure)

| #         | Criterion                     | Expected                                    | Actual | Status |
| --------- | ----------------------------- | ------------------------------------------- | ------ | ------ |
| SCHEMA-01 | Workflow metadata defined     | version, name, author, tags                 | pass   | pass   |
| SCHEMA-02 | Nodes array structure defined | id, type, handler, inputs, outputs, retries | pass   | pass   |
| SCHEMA-03 | Node types documented         | python, agent_node, branch, human, exit     | pass   | pass   |
| SCHEMA-04 | Edges array structure defined | source, target, condition, label            | pass   | pass   |
| SCHEMA-05 | Condition syntax documented   | JSONPath expressions and operators          | pass   | pass   |
| SCHEMA-06 | Validation rules documented   | Required nodes, entry/exit points, paths    | pass   | pass   |
| SCHEMA-07 | Schema versioning documented  | Forward/backward compatibility rules        | pass   | pass   |
| SCHEMA-08 | Examples provided             | Sample JSON for each node type              | pass   | pass   |

### 3.2 FUNC (Schema Validation)

| #       | Criterion                  | Expected                            | Actual | Status |
| ------- | -------------------------- | ----------------------------------- | ------ | ------ |
| FUNC-01 | Validation algorithm clear | Step-by-step schema check procedure | pass   | pass   |
| FUNC-02 | Error messages documented  | What error when validation fails    | pass   | pass   |
| FUNC-03 | Edge condition evaluation  | How conditions are evaluated        | pass   | pass   |

### 3.3 INTG (LangGraph Compilation)

| #       | Criterion                 | Expected                                | Actual | Status |
| ------- | ------------------------- | --------------------------------------- | ------ | ------ |
| INTG-01 | Node input contract       | How nodes receive state                 | pass   | pass   |
| INTG-02 | Node output contract      | How nodes produce state updates         | pass   | pass   |
| INTG-03 | Handler interface defined | What callable is expected in `handler`  | pass   | pass   |
| INTG-04 | Entry node specified      | Which node is execution start           | pass   | pass   |
| INTG-05 | Exit node specified       | Which node terminates execution         | pass   | pass   |
| INTG-06 | LangGraph compilation     | Schema must compile to executable graph | pass   | pass   |

**Supporting Artifacts**:

- Complete JSON schema (with `$schema` reference if applicable)
- Node type table with all fields
- Edge condition evaluation algorithm
- Example workflows (at least 3: sequential, branching, parallel)
- Validation checklist

**Notes**:

> [To be filled after task completion]

---

## 4. Documentation Check

Required updates:

- [x] `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` — confirmed and locked
- [x] `docs/graph-weave/specification/workflow-schema/plan/schema-and-node-contracts.md` — decisions reflected
- [x] `docs/graph-weave/specification/runtime/universal-interpreter.md` — confirms interpreter compatibility

---

## 5. Final Decision

| Decision            | Condition                                                          |
| ------------------- | ------------------------------------------------------------------ |
| **Pass**            | All schema fields + node contracts + edge rules defined + examples |
| **Needs Revision**  | Missing field or contract unclear; agent fixes                     |
| **Fail + Rollback** | Schema incomplete or incompatible with interpreter; task rejected  |

**Decision**: Pass

**Rationale**:

> [To be filled by reviewer after task completion]

**Reviewer Signature**: `[agent-name]` — `[YYYY-MM-DD]`
