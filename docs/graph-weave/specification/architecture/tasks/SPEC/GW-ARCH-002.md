# GW-ARCH-002: Define multi-tenant boundaries and isolation levels

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : Critical
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Multi-tenant architecture
- **Feature** : Establish tenant isolation rules and scoping
- **Rationale** : Core security boundary; must be locked before MOCK phase to prevent data leakage and ensure GraphWeave can serve multiple tenants safely

---

### Input

- **Data / Files** : `[[../specification/architecture/multi-tenant.md]]`, `[[../specification/architecture/plan/tenant-isolation-and-scoping.md]]`, `[[../specification/data/plan/tenant-aware-control-flags.md]]`
- **Dependencies** : GW-ARCH-001 (system boundaries must be defined first)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define tenant scope (top-level isolation unit)
  - Define workflow scope (tenant-level isolation within requests)
  - Define thread scope (execution-level isolation within runs)
  - Document blast radius for each scope (what isolation protects against)
  - Map tenant boundaries to API routes and database queries
  - Record tenant_id routing rules for gateway and runtime

- **Out of Scope**:
  - Implementation details of tenant lookup or routing code
  - Database schema (tenant_id column placement, etc.)
  - Redis namespace key design (see GW-DATA-001)
  - Role-based access control (tenant-aware control flags scope)

- **Max Increment**: Complete tenant isolation architecture specification

---

### Approach

1. Synthesize multi-tenant and tenant-isolation-and-scoping docs
2. Define three-level nesting: Tenant → Workflow → Thread
3. Document what each boundary protects and failure modes if violated
4. Map boundaries to API contract (where tenant_id comes from, how it flows)
5. Define routing rules for gateway and runtime enforcement

**Files to Modify/Create**:

- `docs/graph-weave/specification/architecture/multi-tenant.md` — Update with complete three-level isolation model
- `docs/graph-weave/specification/architecture/plan/tenant-isolation-and-scoping.md` — Confirm decisions match spec

---

### Expected Output

- **Deliverable** : Multi-tenant isolation specification with routing rules and failure modes
- **Format** : Markdown with tables and ASCII flow diagrams
- **Example** :

```
Isolation Level | Scope | Protection | Enforcement
Tenant          | API   | Data leakage across customers | Auth + routing
Workflow        | Request | State pollution between runs | Queue isolation
Thread          | Execution | Control-flow interference | Memory/stack isolation
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-ARCH-002.md]]`

---

### References

- `[[../specification/architecture/multi-tenant.md]]` — Living spec for multi-tenant boundaries; must be consistent
- `[[../specification/architecture/plan/tenant-isolation-and-scoping.md]]` — Decisions on isolation rules; must reflect in output
- `[[../specification/data/plan/tenant-aware-control-flags.md]]` — Data-layer tenant scoping rules; cross-reference
