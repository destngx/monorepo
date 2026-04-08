## 1. Objective

- What: Detect repeated routing behavior and stop runaway workflow loops.
- Why: Prevent the orchestrator from cycling without new information.
- Who: Runtime engineers and workflow authors.

## Traceability

- FR-RUNTIME-030: Stagnation detection must use a sliding window.
- FR-RUNTIME-031: Repeated directives, repeated objectives, and semantic repetition can all trigger stagnation.
- FR-RUNTIME-032: Stagnation must force a safe exit path.

## 2. Scope

- In scope: repeated directives, repeated objectives, semantic repetition, and force-exit behavior.
- Out of scope: user-facing research summaries and external provider behavior.

## 3. Specification

- Repetition must be judged over a sliding window.
- Exact directive repetition, repeated objectives, and semantic similarity can all trip stagnation.
- Once stagnation is detected, the workflow must route to the guardrail/exit path.
- The default threshold is 3, but it must be configurable per workflow.
- The docs should distinguish exact repetition from semantic repetition.
- NFR: detection must be lightweight enough to run on every orchestrator turn.
- A meta-message must be injected before routing to the guardrail path.
- Stagnation detection must not mutate workflow state beyond the meta-message and guardrail route.

## 4. Technical Plan

- Record recent orchestrator outputs and compare them across a configurable window.
- Increment a stagnation counter when repetition is detected.
- Exit after the counter reaches the configured threshold.
- Preserve recent intent history only as long as needed for comparison.
- Keep the threshold semantics configurable without changing the overall safe-exit behavior.
- Keep the detector separate from the circuit breaker so each concern has one responsibility.

## 5. Tasks

- [ ] Persist recent routing decisions for comparison.
- [ ] Compare directive, objective, and semantic similarity patterns.
- [ ] Route to the guardrail when the threshold is exceeded.
- [ ] Add tests for exact, objective, and semantic repetition.
- [ ] Inject a stagnation meta-message before the guardrail path.

## 6. Verification

- Given repeated routing directives, when the window threshold is met, then stagnation must be detected.
- Given repeated objectives, when the same intent appears, then the workflow must not keep looping.
- Given semantically similar outputs, when similarity exceeds the threshold, then the guardrail path must be taken.
- Given a workflow configured with a custom threshold, when the limit is reached, then the safe-exit path must remain the same.
- Given stagnation is detected, when the exit path is taken, then a meta-message must be injected into context.

flowchart TD
A[Receive routing_directive] --> B{Action == CALL_SUBAGENT?}
B -- Yes --> C[Append Intent to stagnation_history]
C --> D{Count(Intent) >= Threshold?}
D -- Yes --> E[Set stagnation_detected = True]
E --> F[Inject Meta-Message to Context]
F --> G[Route to output_guardrail]
D -- No --> H[Route to SubAgent]
B -- No --> I[Standard Routing]

````

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
````
