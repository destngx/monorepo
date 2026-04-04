```mermaid
flowchart TD
    A[Receive routing_directive] --> B{Action == CALL_SUBAGENT?}
    B -- Yes --> C[Append Intent to stagnation_history]
    C --> D{Count(Intent) >= Threshold?}
    D -- Yes --> E[Set stagnation_detected = True]
    E --> F[Inject Meta-Message to Context]
    F --> G[Route to output_guardrail]
    D -- No --> H[Route to SubAgent]
    B -- No --> I[Standard Routing]
```

Important behavior:

- Stagnation is not just identical routing; it can also be repeated objectives or semantically similar outputs.
- The detector uses a sliding window so short noise does not trigger the exit path too early.
- When stagnation trips, the workflow emits a meta-message and routes to the guardrail instead of looping indefinitely.

```stateDiagram-v2
    [*] --> Monitor: Step N

    Monitor --> Record: Store routing decision

    Record --> CheckWindow: Window size >=3?
    CheckWindow --> Monitor: No (continue)

    CheckWindow --> Compare: Yes
    Compare --> Pattern1: Exact match?
    Compare --> Pattern2: Same objective?
    Compare --> Pattern3: Semantic similarity?

    Pattern1 --> Stagnation: Any pattern detected
    Pattern2 --> Stagnation
    Pattern3 --> Stagnation

    Stagnation --> Action: Increment counter
    Action --> Counter{Count >=3?}
    Counter -->|No| Monitor: Reset window
    Counter -->|Yes| FORCE_EXIT: Terminate workflow

    FORCE_EXIT --> [*]
```
