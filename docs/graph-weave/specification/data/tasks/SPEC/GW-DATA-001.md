# GW-DATA-001: Define Redis namespace and key structure

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : Critical
- **Status** : Done
- **Estimated Effort**: L
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : State and cache management
- **Feature** : Establish Redis namespace hierarchy and key naming conventions
- **Rationale** : Storage layout must be locked before MOCK to prevent schema thrashing and ensure tenant isolation is enforced at data layer

---

### Input

- **Data / Files** : `[[../specification/data/redis-namespace-design.md]]`, `[[../specification/data/plan/redis-namespace-layout-and-scope.md]]`, `[[../specification/data/README.md]]`
- **Dependencies** : GW-ARCH-002 (tenant isolation rules), GW-ARCH-003 (request lifecycle)
- **External Systems**: Redis (target storage system)

---

### Scope

- **In Scope** :
  - Define Redis key structure by scope: tenant, workflow, execution, checkpoint
  - Document namespace hierarchy to enforce tenant isolation
  - Define TTL intent for each key type
  - Document key naming conventions (prefixes, separators, encoding)
  - Map Redis operations to tenant/workflow boundaries
  - Define what keys are readable/writable at each scope level

- **Out of Scope**:
  - Redis cluster topology or replication strategy
  - Connection pooling or performance tuning
  - Actual value serialization format (see GW-DATA-002)
  - Cache eviction policies (ops concern)

- **Max Increment**: Complete Redis namespace specification

---

### Approach

1. Synthesize redis-namespace-design and plan docs
2. Define namespace hierarchy: `{env}:{tenant_id}:{type}:{resource_id}:{field}`
3. Document each key type (workflow, run, checkpoint, control flags)
4. Define scope isolation (tenant keys invisible to other tenants)
5. Define TTL rules by key type (ephemeral vs. persistent)

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/redis-namespace-design.md` — Update with complete namespace specification
- `docs/graph-weave/specification/data/plan/redis-namespace-layout-and-scope.md` — Confirm namespace decisions

---

### Expected Output

- **Deliverable** : Redis namespace design with key patterns, scopes, and TTL rules
- **Format** : Markdown with tables and key examples
- **Example** :

```
Key Pattern | Scope | TTL | Purpose
gw:t{tid}:w:{wid} | Tenant | 7d | Workflow definition
gw:t{tid}:r:{rid}:state | Workflow | 30m | Current execution state
gw:t{tid}:r:{rid}:chk:{n} | Workflow | 7d | Checkpoint {n}
gw:t{tid}:r:{rid}:active | Workflow | 5m | Is run active
gw:t{tid}:kill:{rid} | Tenant | 1m | Kill signal
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-DATA-001.md]]`

---

### References

- `[[../specification/data/redis-namespace-design.md]]` — Canonical namespace design; ensure alignment
- `[[../specification/data/plan/redis-namespace-layout-and-scope.md]]` — Decisions on key structure; must be reflected
- `[[../specification/architecture/plan/tenant-isolation-and-scoping.md]]` — Tenant boundaries; must align with Redis scoping
