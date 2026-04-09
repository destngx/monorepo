# GW-WF-102: Allow multiple phase labels

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Workflow schema documentation
- **Feature** : Multi-label phase support for workflow requirements in the mock app
- **Rationale** : MOCK needs one requirement to carry more than one label when it spans phases

---

### Input

- **Data / Files** : `[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]]`, `[[../../workflow-schema/README.md]]`, `[[../../workflow-schema/plan/schema-and-node-contracts.md]]`
- **Dependencies** : GW-WF-101
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Preserve multiple labels where requirements span more than one phase
  - Keep the prompt-driven model wording stable in the mock app docs

- **Out of Scope**:
  - Schema implementation details
  - Runtime execution changes
  - Skill loading implementation

- **Max Increment**: One working mock multi-label rule

---

### Approach

1. Confirm the label convention in the spec README
2. Keep workflow requirements tagged with multiple phase labels when needed

**Files to Modify/Create**:

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` — source of truth for workflow labels
- `docs/graph-weave/specification/workflow-schema/verification/MOCK/VERIFY-GW-WF-102-SCHEMA.md` — verify multi-label support is documented
- `docs/graph-weave/specification/workflow-schema/verification/MOCK/VERIFY-GW-WF-102-DOC.md` — verify the multi-label rule stays clear

---

### Expected Output

- **Deliverable** : Working mock multi-label rule
- **Format** : Markdown
- **Example** : `FR-WF-001 [MOCK,MVP,FULL]`

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-102-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-WF-102-DOC.md]]

---

### References

[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]] - Canonical workflow schema document; labels must remain consistent with its requirement list.
