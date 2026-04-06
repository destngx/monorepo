```mermaid
## 1. Objective

- What: Detect repeated routing behavior and stop runaway workflow loops.
- Why: Prevent the orchestrator from cycling without new information.
- Who: Runtime engineers and workflow authors.

## 2. Scope

- In scope: repeated directives, repeated objectives, semantic repetition, and force-exit behavior.
- Out of scope: user-facing research summaries and external provider behavior.

## 3. Specification

- Repetition must be judged over a sliding window.
- Exact directive repetition, repeated objectives, or semantic similarity can all trip stagnation.
- Once stagnation is detected, the workflow must route to the guardrail/exit path.

## 4. Technical Plan

- Record recent orchestrator outputs and compare them across a configurable window.
- Increment a stagnation counter when repetition is detected.
- Exit after the counter reaches the configured threshold.

## 5. Tasks

- [ ] Persist recent routing decisions for comparison.
- [ ] Compare directive, objective, and semantic similarity patterns.
- [ ] Route to the guardrail when the threshold is exceeded.

## 6. Verification

- Given repeated routing directives, when the window threshold is met, then stagnation must be detected.
- Given repeated objectives, when the same intent appears, then the workflow must not keep looping.
- Given semantically similar outputs, when similarity exceeds the threshold, then the guardrail path must be taken.

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
