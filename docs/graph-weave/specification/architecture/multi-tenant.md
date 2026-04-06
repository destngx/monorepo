## 1. Objective

- What: Define tenant isolation for workflow state, skills, and kill switches.
- Why: Prevent one tenant from reading or affecting another tenant's execution.
- Who: Multi-tenant platform operators and runtime engineers.

## 2. Scope

- In scope: tenant IDs, thread IDs, workflow namespaces, skill namespaces, and kill-switch scoping.
- Out of scope: cross-tenant data sharing and shared mutable execution state.

## 3. Specification

- Every request must carry a tenant_id and thread_id.
- Checkpoints must remain thread-scoped.
- Redis namespaces must be tenant-aware for workflows, skills, and kill switches.
- Active threads must be trackable per tenant for cancellation and audit.

## 4. Technical Plan

- Encode tenant identity into every runtime key.
- Store state and schema data in separate Redis namespaces.
- Ensure kill-switch keys can target tenant, workflow, or thread blast radii.

## 5. Tasks

- [ ] Encode tenant/thread scope into workflow and checkpoint keys.
- [ ] Separate tier-1 and tier-2 skill namespaces.
- [ ] Add tenant-scoped active-thread tracking and kill-switch controls.

## 6. Verification

- Given two tenants, when they execute simultaneously, then their state and checkpoints must remain isolated.
- Given a kill switch is set for one tenant, when another tenant runs, then it must not be interrupted.
- Given active threads are listed, when a tenant is inspected, then only that tenant's threads should appear.

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
