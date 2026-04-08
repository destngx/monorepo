# GW-DATA-003: Specify tenant-aware control flags and blast radius

### Metadata

- **Phase ID** : SPEC
- **Risk Level** : Critical
- **Status** : Done
- **Estimated Effort**: M
- **Assigned Agent** : Hephaestus

---

### Context

- **Bounded Context** : Multi-tenant runtime controls
- **Feature** : Define control flags for kill-switch, rate limits, feature flags scoped by tenant
- **Rationale** : Control plane must be tenant-aware to prevent one tenant from affecting others; must be locked before MVP to ensure safety

---

### Input

- **Data / Files** : `[[../specification/data/plan/tenant-aware-control-flags.md]]`, `[[../specification/architecture/plan/tenant-isolation-and-scoping.md]]`, `[[../specification/data/plan/redis-namespace-layout-and-scope.md]]`
- **Dependencies** : GW-ARCH-002 (tenant isolation), GW-DATA-001 (Redis namespace)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Define control flag types: kill-switch, rate limit, feature flags, quota
  - Document scope for each flag: tenant, workflow, thread level
  - Define blast radius rules (what is affected if flag is set)
  - Document flag atomicity (how flags interact with concurrent runs)
  - Map flags to Redis keys using namespace rules
  - Define flag TTL and cleanup behavior

- **Out of Scope**:
  - Implementation of flag checking logic
  - Admin API for setting flags
  - Monitoring or alerting on flags
  - Rate limit enforcement algorithm

- **Max Increment**: Complete control flags specification

---

### Approach

1. Synthesize tenant-aware-control-flags plan and tenant isolation docs
2. Define flag types and their scopes
3. Document blast radius: kill-switch affects tenant, rate-limit affects workflow, feature-flag affects all
4. Define flag structure in Redis (values, TTL, atomicity)
5. Define how flags are checked during execution

**Files to Modify/Create**:

- `docs/graph-weave/specification/data/plan/tenant-aware-control-flags.md` — Confirm flag design and scopes

---

### Expected Output

- **Deliverable** : Tenant-aware control flags specification with blast radius rules
- **Format** : Markdown with decision table
- **Example** :

```
Flag Type | Scope | Key Pattern | Blast Radius | TTL | Check Point
kill-switch | Tenant | gw:t{tid}:kill | All workflows in tenant | 1m | Pre-node
rate-limit | Workflow | gw:t{tid}:w{wid}:ratelimit | Requests in workflow | 5m | Pre-request
feature-flag | Tenant | gw:t{tid}:feature:{name} | All runs in tenant | 24h | Interpreter init
quota | Tenant | gw:t{tid}:quota | All runs in tenant | 7d | Post-execution
```

---

### Verification Criteria

See: `[[../SPEC/VERIFY-GW-DATA-003.md]]`

---

### References

- `[[../specification/data/plan/tenant-aware-control-flags.md]]` — Flag decisions; must be reflected in spec
- `[[../specification/architecture/plan/tenant-isolation-and-scoping.md]]` — Tenant boundaries; flags must respect scopes
- `[[./GW-DATA-001.md]]` — Redis namespace; flags must use defined patterns
- `[[./GW-RUNTIME-002.md]]` — Kill-switch is a control flag; must integrate
