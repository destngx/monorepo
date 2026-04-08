# VERIFY-GW-WF-001: Lock workflow JSON schema and node contracts

> **Linked Task** : GW-WF-001 — `docs/graph-weave/specification/workflow-schema/tasks/SPEC/GW-WF-001.md`
> **Verification Types** : SCHEMA, FUNC, INTG
> **Phase ID** : SPEC
> **Risk Level** : Critical
> **Reviewer** : TBD
> **Verified On** : [pending]
> **Overall Status** : Pending

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

| Criterion                   | Expected                               | Status  |
| --------------------------- | -------------------------------------- | ------- |
| Single deliverable          | One workflow schema specification      | pending |
| No workflow editor/UI       | Defers UI to frontend work             | pending |
| No execution semantics      | References GW-RUNTIME-\* for execution | pending |
| No schema migration tooling | Defers to tooling phase                | pending |
| No example workflows        | Defers to GW-SKILL-\* examples         | pending |

---

## 3. Type-Specific Criteria

### 3.1 SCHEMA (JSON Structure)

| #         | Criterion                     | Expected                                    | Actual | Status  |
| --------- | ----------------------------- | ------------------------------------------- | ------ | ------- |
| SCHEMA-01 | Workflow metadata defined     | version, name, author, tags                 |        | pending |
| SCHEMA-02 | Nodes array structure defined | id, type, handler, inputs, outputs, retries |        | pending |
| SCHEMA-03 | Node types documented         | python, agent_node, branch, human, exit     |        | pending |
| SCHEMA-04 | Edges array structure defined | source, target, condition, label            |        | pending |
| SCHEMA-05 | Condition syntax documented   | JSONPath expressions and operators          |        | pending |
| SCHEMA-06 | Validation rules documented   | Required nodes, entry/exit points, paths    |        | pending |
| SCHEMA-07 | Schema versioning documented  | Forward/backward compatibility rules        |        | pending |
| SCHEMA-08 | Examples provided             | Sample JSON for each node type              |        | pending |

### 3.2 FUNC (Schema Validation)

| #       | Criterion                  | Expected                            | Actual | Status  |
| ------- | -------------------------- | ----------------------------------- | ------ | ------- |
| FUNC-01 | Validation algorithm clear | Step-by-step schema check procedure |        | pending |
| FUNC-02 | Error messages documented  | What error when validation fails    |        | pending |
| FUNC-03 | Edge condition evaluation  | How conditions are evaluated        |        | pending |

### 3.3 INTG (LangGraph Compilation)

| #       | Criterion                 | Expected                                | Actual | Status  |
| ------- | ------------------------- | --------------------------------------- | ------ | ------- |
| INTG-01 | Node input contract       | How nodes receive state                 |        | pending |
| INTG-02 | Node output contract      | How nodes produce state updates         |        | pending |
| INTG-03 | Handler interface defined | What callable is expected in `handler`  |        | pending |
| INTG-04 | Entry node specified      | Which node is execution start           |        | pending |
| INTG-05 | Exit node specified       | Which node terminates execution         |        | pending |
| INTG-06 | LangGraph compilation     | Schema must compile to executable graph |        | pending |

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

- [ ] `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` — confirmed and locked
- [ ] `docs/graph-weave/specification/workflow-schema/plan/schema-and-node-contracts.md` — decisions reflected
- [ ] `docs/graph-weave/specification/runtime/universal-interpreter.md` — confirms interpreter compatibility

---

## 5. Final Decision

| Decision            | Condition                                                          |
| ------------------- | ------------------------------------------------------------------ |
| **Pass**            | All schema fields + node contracts + edge rules defined + examples |
| **Needs Revision**  | Missing field or contract unclear; agent fixes                     |
| **Fail + Rollback** | Schema incomplete or incompatible with interpreter; task rejected  |

**Decision**: Pending

**Rationale**:

> [To be filled by reviewer after task completion]

**Reviewer Signature**: `[agent-name]` — `[YYYY-MM-DD]`
