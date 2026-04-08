## 1. Objective

- What: Define tenant isolation for workflow state, skills, and kill switches.
- Why: Prevent one tenant from reading or affecting another tenant's execution.
- Who: Multi-tenant platform operators and runtime engineers.

## Traceability

- FR-MT-001: Every request must carry tenant, workflow, and thread scope.
- FR-MT-002: State and kill-switch behavior must remain isolated by tenant.
- FR-MT-003: Active threads must be auditable per tenant.

## 2. Scope

- In scope: tenant IDs, thread IDs, workflow namespaces, skill namespaces, and kill-switch scoping.
- Out of scope: cross-tenant data sharing and shared mutable execution state.

## 3. Specification

- Every request must carry a tenant_id and thread_id.
- Every request must carry tenant_id, workflow_id, and thread_id together so scope is explicit at ingress.
- Checkpoints must remain thread-scoped.
- Redis namespaces must be tenant-aware for workflows, skills, and kill switches.
- Active threads must be trackable per tenant for cancellation and audit.
- The authoritative tenant model is tenant + workflow + thread.
- Redis key examples are guidelines; implementations may refine names if they preserve tenant/workflow/thread isolation.
- Workflow execution, skill caches, and kill switches must all respect tenant scoping.
- Routing decisions must preserve the current tenant context through gateway, runtime, and storage calls.
- A tenant-scoped kill switch must never affect threads outside its tenant.

## 4. Technical Plan

- Encode tenant identity into every runtime key.
- Store state and schema data in separate Redis namespaces.
- Ensure kill-switch keys can target tenant, workflow, or thread blast radii.
- Keep per-tenant auditability explicit in every runtime store interaction.
- Do not share mutable execution state across tenants.
- Keep active-thread listings and cache namespaces tenant-scoped.
- Enforce a stable routing chain: gateway identifies scope, runtime preserves it, storage partitions by it.
- Treat blast radius as a first-class property for every control flag.

## 5. Tasks

- [ ] Encode tenant/thread scope into workflow and checkpoint keys.
- [ ] Separate tier-1 and tier-2 skill namespaces.
- [ ] Add tenant-scoped active-thread tracking and kill-switch controls.
- [ ] Document the blast-radius rules for each kill-switch scope.

## 6. Verification

- Given two tenants, when they execute simultaneously, then their state and checkpoints must remain isolated.
- Given a kill switch is set for one tenant, when another tenant runs, then it must not be interrupted.
- Given active threads are listed, when a tenant is inspected, then only that tenant's threads should appear.
- Given a workflow scoped request, when it is executed, then the runtime must be able to identify tenant, workflow, and thread together.

```mermaid
graph TB
    subgraph "Request isolation"
        A[Tenant A request] --> ACFG[tenant_id=A, thread_id=A1]
        B[Tenant B request] --> BCFG[tenant_id=B, thread_id=B1]
        ACFG --> ASTATE[(Checkpoint A1)]
        BCFG --> BSTATE[(Checkpoint B1)]
    end

    subgraph "Redis namespaces"
        R1[workflow:{tenant}:{workflow}:{version}]
        R2[skills:tier1:{tenant}:{skill_id}]
        R3[skills:tier2:{tenant}:{skill_id}]
        R4[graphweave:circuit_breaker:{scope}:{id}:kill]
        R5[active_threads:{tenant}]
    end

    ACFG --> R1
    ACFG --> R2
    ACFG --> R4
    BCFG --> R1
    BCFG --> R3
    BCFG --> R5
```
