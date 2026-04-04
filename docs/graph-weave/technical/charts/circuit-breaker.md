```mermaid
stateDiagram-v2
    [*] --> Closed: workflow starts

    Closed --> Closed: normal execution
    Closed --> Open: kill switch set<br/>(tenant / workflow / thread)

    Open --> Interrupted: watchdog calls interrupt()
    Interrupted --> Draining: checkpoint + stop new work
    Draining --> HalfOpen: TTL expires (300s)

    HalfOpen --> Closed: manual override or health check passes
    HalfOpen --> Open: kill flag still present

    note right of Open
        Redis keys:
        graphweave:circuit_breaker:*:kill
        TTL: 300s
    end note
```

Operational notes:

- A kill switch can be scoped at tenant, workflow, or thread level.
- Interrupt happens at the watchdog boundary so the graph can stop cleanly instead of corrupting state.
- The half-open state is useful for re-checking health after TTL expiry instead of permanently leaving a workflow disabled.
