# VERIFY-GW-DATA-002: Checkpoint and compiled graph storage

> **Linked Task** : GW-DATA-002 — `docs/graph-weave/specification/data/tasks/SPEC/GW-DATA-002.md`
> **Verification Types** : SCHEMA, INTG, QUALITY
> **Phase ID** : SPEC
> **Risk Level** : High
> **Reviewer** : TBD
> **Verified On** : [pending]
> **Overall Status** : Pending

---

## 1. Traceability

The specification must reference:

- `docs/graph-weave/specification/data/plan/compiled-graph-and-checkpoint-storage.md`
- `docs/graph-weave/specification/runtime/universal-interpreter.md`

**Evidence**: Cross-references confirm alignment with source documents.

---

## 2. Scope Compliance

Verify task produced one deliverable: checkpoint and cache specification.

| Criterion               | Expected                                       | Status  |
| ----------------------- | ---------------------------------------------- | ------- |
| Single deliverable      | One storage specification                      | pending |
| No compilation logic    | References workflow schema but doesn't compile | pending |
| No serialization format | References but defers value encoding           | pending |
| No database schema      | References but defers storage details          | pending |
| No retention policies   | Ops concern, not included                      | pending |

---

## 3. Type-Specific Criteria

### 3.1 SCHEMA (Storage Format)

| #         | Criterion                     | Expected                                        | Actual | Status  |
| --------- | ----------------------------- | ----------------------------------------------- | ------ | ------- |
| SCHEMA-01 | Checkpoint structure defined  | run_id, version, last_node, timestamp, state    |        | pending |
| SCHEMA-02 | State fields documented       | What interpreter state is captured              |        | pending |
| SCHEMA-03 | Compiled graph schema defined | What is cached (graph structure only, not code) |        | pending |
| SCHEMA-04 | Key patterns use GW-DATA-001  | Keys follow namespace design                    |        | pending |

### 3.2 INTG (Retrieval & Lifecycle)

| #       | Criterion                   | Expected                                       | Actual | Status  |
| ------- | --------------------------- | ---------------------------------------------- | ------ | ------- |
| INTG-01 | Resume semantics clear      | How to restore from checkpoint                 |        | pending |
| INTG-02 | Cache strategy documented   | Compiled graphs cached in Redis or re-fetched  |        | pending |
| INTG-03 | TTL rules documented        | Checkpoint TTL vs. cache TTL                   |        | pending |
| INTG-04 | Interpreter state alignment | Checkpoint captures all fields from GW-RUNTIME |        | pending |

### 3.3 QUALITY (Structure Clarity)

| #          | Criterion              | Expected                                        | Actual | Status  |
| ---------- | ---------------------- | ----------------------------------------------- | ------ | ------- |
| QUALITY-01 | Completeness verified  | Checkpoint captures everything needed to resume |        | pending |
| QUALITY-02 | Documentation is clear | No ambiguity in storage format                  |        | pending |

**Supporting Artifacts**:

- JSON schema for checkpoint structure
- Resume flow diagram
- Cache strategy decision (Redis vs. fetch)

**Notes**:

> [To be filled after task completion]

---

## 4. Documentation Check

Required updates:

- [ ] `docs/graph-weave/specification/data/plan/compiled-graph-and-checkpoint-storage.md` — decisions confirmed
- [ ] Checkpoint captures all fields from interpreter state (GW-RUNTIME-001)

---

## 5. Final Decision

| Decision            | Condition                                                              |
| ------------------- | ---------------------------------------------------------------------- |
| **Pass**            | All SCHEMA + FUNC criteria met + resume is possible from checkpoint    |
| **Needs Revision**  | Missing state fields or cache strategy unclear; agent fixes            |
| **Fail + Rollback** | Checkpoint can't restore state; critical data loss risk; task rejected |

**Decision**: Pending

**Rationale**:

> [To be filled by reviewer after task completion]

**Reviewer Signature**: `[agent-name]` — `[YYYY-MM-DD]`
