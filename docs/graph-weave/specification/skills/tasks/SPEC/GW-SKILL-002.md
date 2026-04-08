# GW-SKILL-002: Specify three-level skill loading and progressive disclosure

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : High
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Skill runtime loading
- **Feature** : Define progressive skill loading to balance discovery with performance
- **Rationale** : Loading strategy affects latency and security; must be locked before MVP to prevent mid-flight changes

---

### Input

- **Data / Files** : `[[../specification/skills/skill-loading-flow.md]]`, `[[../specification/skills/llm-skills-architecture.md]]`, `[[../specification/skills/plan/skill-loading-and-packaging.md]]`, `[[../specification/runtime/plan/universal-interpreter-and-skill-loading.md]]`
- **Dependencies** : GW-SKILL-001 (registry must be defined), GW-RUNTIME-001 (interpreter interface)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define three loading levels: Level 1 (pre-loaded), Level 2 (on-demand), Level 3 (lazy at execution)
  - Document when each level is triggered
  - Define loading boundary (interpreter pre-receives skills vs. requests them)
  - Document skill availability guarantees at each level
  - Define error behavior if skill not available
  - Map loading to request lifecycle and checkpoints

- **Out of Scope**:
  - Skill code implementation
  - Skill caching or storage
  - Performance optimization
  - Distribution or deployment of skills

- **Max Increment**: Complete skill loading specification

---

### Approach

1. Synthesize skill-loading-flow, llm-skills-architecture, and plan docs
2. Define Level 1: core skills pre-loaded at startup
3. Define Level 2: user-defined skills loaded per-workflow
4. Define Level 3: runtime-discovered skills loaded on-demand during execution
5. Document when each level is activated and availability guarantees

**Files to Modify/Create**:

- `docs/graph-weave/specification/skills/skill-loading-flow.md` — Update with three-level loading flow
- `docs/graph-weave/specification/skills/plan/skill-loading-and-packaging.md` — Confirm loading decisions
- `docs/graph-weave/specification/runtime/plan/universal-interpreter-and-skill-loading.md` — Confirm interpreter doesn't do loading

---

### Expected Output

- **Deliverable** : Three-level skill loading specification
- **Format** : Markdown with state diagram and timeline
- **Example** :

```
Level | When | Scope | Availability | Latency
1     | Startup | All tenants | Guaranteed | Pre-computed
2     | Submit request | Per-workflow | If in registry | +load time
3     | During execution | Per-run | If discovered | Lazy, nullable
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-SKILL-002.md]]`

---

### References

- `[[../specification/skills/skill-loading-flow.md]]` — Canonical loading flow; must align with three-level model
- `[[../specification/skills/plan/skill-loading-and-packaging.md]]` — Loading decisions; must be reflected in spec
- `[[./GW-RUNTIME-001.md]]` — Interpreter receives pre-loaded skills; must confirm interface
- `[[./GW-ARCH-003.md]]` — Request lifecycle; loading must integrate with lifecycle
