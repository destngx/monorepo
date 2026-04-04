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
