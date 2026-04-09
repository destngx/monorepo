# GW-WF-101: Add single phase labels

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Pending
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Workflow schema documentation
- **Feature** : Single phase labels for workflow requirements in the mock app
- **Rationale** : MOCK needs one clear label per requirement before multi-label requirements are added

---

### Input

- **Data / Files** : `[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]]`, `[[../../workflow-schema/README.md]]`, `[[../../workflow-schema/plan/schema-and-node-contracts.md]]`
- **Dependencies** : GW-WF-001
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Keep one phase label visible on workflow requirements
  - Keep the prompt-driven model wording stable in the mock app docs

- **Out of Scope**:
  - Multi-label support
  - Schema implementation details
  - Runtime execution changes

- **Max Increment**: One working mock single-label rule

---

### Approach

1. Confirm the label convention in the spec README
2. Keep workflow requirements tagged with phase labels
3. Ensure the docs continue to present prompt-driven autonomy clearly

**Files to Modify/Create**:

- `docs/graph-weave/specification/workflow-schema/WORKFLOW_JSON_SPEC.md` — source of truth for workflow labels
- `docs/graph-weave/specification/workflow-schema/verification/MOCK/VERIFY-GW-WF-101-SCHEMA.md` — verify single-label support is documented
- `docs/graph-weave/specification/workflow-schema/verification/MOCK/VERIFY-GW-WF-101-DOC.md` — verify the mock-phase wording stays clear

---

### Expected Output

- **Deliverable** : Working mock single-label rule
- **Format** : Markdown
- **Example** : `FR-WF-001 [MOCK]`

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-WF-101-SCHEMA.md]]
[[../../verification/MOCK/VERIFY-GW-WF-101-DOC.md]]

---

### References

[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]] - Canonical workflow schema document; labels must remain consistent with its requirement list.
