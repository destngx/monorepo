# GW-SKILL-001: Specify skill registry and metadata contracts

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Skill discovery and loading
- **Feature** : Define skill registry structure and metadata format
- **Rationale** : Registry is the contract between workflow definitions and runtime; must be locked before MOCK to prevent discovery bugs

---

### Input

- **Data / Files** : `[[../specification/skills/skill-loading-flow.md]]`, `[[../specification/skills/llm-skills-architecture.md]]`, `[[../specification/skills/plan/skill-registry-and-metadata-contracts.md]]`, `[[../specification/skills/README.md]]`
- **Dependencies** : GW-RUNTIME-001 (interpreter interface defines skill contract)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define skill registry structure (folder layout, discovery mechanism)
  - Define skill metadata format (YAML frontmatter, fields)
  - Define skill discovery rules (Level 1, Level 2, Level 3 disclosure)
  - Document skill versioning and compatibility
  - Define what registry must provide to loader (metadata only, not code)
  - Map registry to storage (filesystem, Redis)

- **Out of Scope**:
  - Skill loading algorithm (see GW-SKILL-002)
  - Skill runtime execution details
  - LLM prompt engineering for skills
  - Skill packaging or distribution

- **Max Increment**: Complete skill registry specification

---

### Approach

1. Synthesize skill-loading-flow, llm-skills-architecture, and plan docs
2. Define registry as metadata index (separate from code)
3. Define skill metadata schema (name, version, inputs, outputs, examples, category)
4. Define three disclosure levels: Level 1 (built-in), Level 2 (user-defined), Level 3 (runtime-discovered)
5. Define discovery mechanism (how interpreter learns what skills exist)

**Files to Modify/Create**:

- `docs/graph-weave/specification/skills/skill-loading-flow.md` — Update with registry discovery flow
- `docs/graph-weave/specification/skills/llm-skills-architecture.md` — Confirm metadata schema
- `docs/graph-weave/specification/skills/plan/skill-registry-and-metadata-contracts.md` — Confirm registry decisions

---

### Expected Output

- **Deliverable** : Skill registry and metadata contract specification
- **Format** : Markdown with YAML examples and folder layout diagram
- **Example** :

```yaml
# skills/registry/math-operations.yaml
name: math-operations
version: 1.0.0
category: utility
inputs:
  - name: operation
    type: string
    values: [add, subtract, multiply, divide]
  - name: operands
    type: array
    items: number
outputs:
  - name: result
    type: number
examples:
  - operation: add
    operands: [2, 3]
    result: 5
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-SKILL-001.md]]`

---

### References

- `[[../specification/skills/skill-loading-flow.md]]` — Canonical loading flow; registry is step 1
- `[[../specification/skills/llm-skills-architecture.md]]` — Skill architecture; registry must support skill contract
- `[[../specification/skills/plan/skill-registry-and-metadata-contracts.md]]` — Registry decisions; must be reflected
- `[[./GW-RUNTIME-001.md]]` — Interpreter expects skills with known interface; registry must provide
