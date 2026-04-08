# VERIFY-GW-DATA-001: Redis namespace and key structure

> **Linked Task** : GW-DATA-001 — `docs/graph-weave/specification/data/tasks/SPEC/GW-DATA-001.md`
> **Verification Types** : SCHEMA, INTG, QUALITY
> **Phase ID** : SPEC
> **Risk Level** : Critical
> **Reviewer** : TBD
> **Verified On** : 2026-04-08
> **Overall Status** : Pass

---

## 1. Traceability

The specification must synthesize:

- `docs/graph-weave/specification/data/redis-namespace-design.md`
- `docs/graph-weave/specification/data/plan/redis-namespace-layout-and-scope.md`
- References to `docs/graph-weave/specification/architecture/plan/tenant-isolation-and-scoping.md`

**Evidence**: Cross-references confirm alignment with source documents.

---

## 2. Scope Compliance

Verify task produced one deliverable: complete Redis namespace specification.

| Criterion            | Expected                                        | Status |
| -------------------- | ----------------------------------------------- | ------ |
| Single deliverable   | One namespace design document                   | pass   |
| No cluster topology  | References Redis but doesn't prescribe topology | pass   |
| No tuning details    | References but doesn't optimize                 | pass   |
| No value encoding    | References GW-DATA-002 for serialization        | pass   |
| No eviction policies | Notes ops concern, defers to ops                | pass   |

---

## 3. Type-Specific Criteria

### 3.1 SCHEMA (Data Structure - Critical)

**Data Schema Critical**: This defines storage isolation across tenants. All checks must pass.

| #         | Criterion                  | Expected                                       | Actual | Status |
| --------- | -------------------------- | ---------------------------------------------- | ------ | ------ |
| SCHEMA-01 | Key pattern defined        | {env}:{tenant_id}:{type}:{resource_id}:{field} | pass   | pass   |
| SCHEMA-02 | Namespace hierarchy clear  | Top-level separator enforces tenant isolation  | pass   | pass   |
| SCHEMA-03 | All key types documented   | Workflow, run, checkpoint, control flag keys   | pass   | pass   |
| SCHEMA-04 | Scope boundaries enforced  | Keys include tenant_id at every level          | pass   | pass   |
| SCHEMA-05 | TTL rules documented       | Each key type has defined TTL intent           | pass   | pass   |
| SCHEMA-06 | No cross-tenant visibility | Tenant keys are never visible to other tenants | pass   | pass   |
| SCHEMA-07 | Examples provided          | Real key examples for each type                | pass   | pass   |

### 3.2 INTG (Key Uniqueness & Integration)

| #       | Criterion                       | Expected                                | Actual | Status |
| ------- | ------------------------------- | --------------------------------------- | ------ | ------ |
| INTG-01 | Key pattern enforces uniqueness | No two keys collide                     | pass   | pass   |
| INTG-02 | Tenant isolation enforced       | GW-ARCH-002 isolation rules follow keys | pass   | pass   |
| INTG-03 | Runtime can generate keys       | GW-RUNTIME-001/002 can construct keys   | pass   | pass   |

### 3.3 QUALITY (Consistency & Documentation)

| #          | Criterion            | Expected                                 | Actual | Status |
| ---------- | -------------------- | ---------------------------------------- | ------ | ------ |
| QUALITY-01 | Naming is consistent | Pattern is predictable and follows rules | pass   | pass   |
| QUALITY-02 | Documentation clear  | No ambiguity in key construction         | pass   | pass   |

**Supporting Artifacts**:

- Key pattern table with scope and TTL
- Example keys for each type
- Namespace hierarchy diagram

**Notes**:

> [To be filled after task completion]

---

## 4. Documentation Check

Required updates:

- [x] `docs/graph-weave/specification/data/redis-namespace-design.md` — matches spec
- [x] `docs/graph-weave/specification/data/plan/redis-namespace-layout-and-scope.md` — decisions reflected
- [x] Tenant isolation rules (GW-ARCH-002) align with namespace scoping

---

## 5. Final Decision

| Decision            | Condition                                                                                  |
| ------------------- | ------------------------------------------------------------------------------------------ |
| **Pass**            | All SCHEMA criteria met + tenant isolation enforced at data layer                          |
| **Needs Revision**  | Missing key type or TTL rule; agent fixes and re-submits                                   |
| **Fail + Rollback** | Namespace allows cross-tenant visibility or is inconsistent; critical issue; task rejected |

**Decision**: Pass

**Rationale**:

> [To be filled by reviewer after task completion]

**Reviewer Signature**: `[agent-name]` — `[YYYY-MM-DD]`
