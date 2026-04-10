## 1. Objective

- What: Define the runtime shape of the universal interpreter graph executor.
- Why: Make execution, routing, and stop conditions explicit while keeping nodes and edges as the workflow source of truth.
- Who: Runtime engineers, workflow authors, and platform operators.

## Traceability

- FR-RUNTIME-040 [MVP,FULL]: A single compiled graph must handle multiple workflows.
- FR-RUNTIME-041 [MVP,FULL]: Level 1 skill frontmatter must always be loaded before routing.
- FR-RUNTIME-042 [MVP,FULL]: Subagent and exit paths must remain isolated and explicit.

## 2. Scope

- In scope: initializer, orchestrator, skill loading, stagnation detection, subagent execution, circuit breaker, and output guardrail.
- Out of scope: provider internals, business-domain logic, and UI presentation.

## 3. Specification

- The same compiled graph must handle multiple workflows through dynamic configuration.
- Interpreter input must include compiled graph, execution state, execution context, and pre-loaded skills.
- Interpreter output must include final state, result, emitted events, and any deferred skill requests.
- Level 1 frontmatter must always be available; Level 2 bodies must be loaded lazily and Level 3 linked files only on demand.
- Subagents must stay isolated and return summarized results.
- Circuit breakers and stagnation detection must be able to force safe exit.
- Output guardrails must run before workflow completion.
- The runtime spec should be treated as authoritative for graph traversal order and safe-exit behavior.
- The interpreter itself must not fetch skills; it receives them from the loading boundary.
- NFR: the interpreter must remain responsive enough for streaming workflows.

## 4. Technical Plan

- Model the runtime as a single reusable graph executor with dynamic branching.
- Treat the interpreter as a pure execution boundary: input state in, output state and events out.
- Route to skill loading only when the orchestrator traversing the graph requests additional context.
- Use the watchdog boundary to interrupt execution safely.
- Return to the orchestrator after subagent completion unless an exit condition is met.
- Keep node naming and control flow aligned with the documented graph contract.
- Allow internal module changes so long as the graph behavior remains the same.

## 5. Tasks

- [ ] Document the canonical graph traversal sequence for execution.
- [ ] Preserve the lazy-load and stagnation branches.
- [ ] Keep safe-exit paths explicit.
- [ ] Map each node to a traceable requirement ID.

## 5.1 Implementation Notes: State Flow and Routing [MVP]

### Execution State Shape

The interpreter maintains execution state as it traverses the graph:

```python
execution_state = {
    "workflow_id": "...",
    "run_id": "...",
    "thread_id": "...",
    "tenant_id": "...",
    "current_node": "research",
    "status": "running",  # queued, validating, pending, running, completed, failed, cancelled
    "node_outputs": {
        "entry": { "topic": "AI ethics", "depth": "thorough" },
        "research_output": { "findings": "...", "confidence": 0.85 },
        # ... accumulated outputs from each node
    },
    "events": [
        { "type": "request.started", "timestamp": "...", "data": {...} },
        # ... immutable append-only event log
    ],
    "checkpoints": [
        { "node_id": "research", "state_snapshot": {...}, "timestamp": "..." },
        # ... persisted checkpoints for recovery
    ],
    "metadata": {
        "start_time": "...",
        "hops": 0,  # Track edges traversed for max_hops enforcement
        "stagnation_count": 0,  # Track repeated nodes for stagnation detection
    }
}
```

### Edge Routing Algorithm (Deterministic)

```python
def route_to_next_node(current_node, workflow_state):
    """
    Deterministic routing based on edge conditions evaluated against state snapshot.
    Returns: (next_node_id, taken_edge_id, condition_value)
    """

    # Snapshot state to ensure immutability during edge evaluation
    state_snapshot = copy.deepcopy(workflow_state["node_outputs"])

    # Get outgoing edges from current node (in definition order)
    outgoing_edges = get_outgoing_edges(current_node, workflow)

    for edge in outgoing_edges:
        if edge.condition is None:
            # Unconditional edge - always true
            return edge.to, edge.id, True

        # Evaluate condition as JSONPath expression
        condition_result = evaluate_jsonpath(
            expression=edge.condition,
            data=state_snapshot
        )

        if condition_result:
            # First true condition is taken
            return edge.to, edge.id, condition_result

    # No edge condition evaluated to true - stagnation
    raise StagnationDetected(current_node, outgoing_edges)
```

### Stagnation Detection and Exit

Stagnation occurs when:

1. A node is visited more than `stagnation_threshold` times consecutively (default: 2).
2. No outgoing edge condition evaluates to true.
3. Workflow exceeds `max_hops` limit.

When stagnation is detected:

1. Emit `state.stagnation` event.
2. Force exit through output guardrail with error status.
3. Return final state and exit event to client.

### Event Emission Pattern

Events are emitted at lifecycle milestones and stored in Redis:

```python
def emit_event(run_id, event_type, data):
    """Store immutable event in Redis append-only log."""
    event = {
        "type": event_type,
        "run_id": run_id,
        "timestamp": datetime.now(timezone.utc).isoformat() + "Z",
        "data": data
    }
    # Append to run event log (cannot be modified or deleted)
    redis.rpush(f"run:{tenant_id}:{run_id}:events", json.dumps(event))
    return event
```

Emission points:

- `request.started` - When execution begins
- `request.validation_started` - Before input validation
- `request.validation_completed` - After validation succeeds
- `node.started` - When entering a node
- `node.completed` - When node produces output
- `node.failed` - When node execution fails
- `node.skipped` - When edge condition causes skip
- `checkpoint.saving` - Before persisting checkpoint
- `checkpoint.saved` - After checkpoint stored
- `state.stagnation` - When loop detected
- `complete` - Final event sent to client

## 6. Verification

- Given a workflow start, when the initializer runs, then the orchestrator must receive a valid state.
- Given a skill request, when the orchestrator selects it, then the loader must fetch Level 2 context only for that skill and open Level 3 files only if needed.
- Given stagnation or a kill flag, when detected, then the graph must exit safely rather than loop.
- Given a supported workflow graph, when the interpreter runs, then it must honor the documented traversal sequence and runtime contract.
- Given edge conditions, when evaluating against state snapshots, then routing must be deterministic (same input state → same next node).
- Given missing node outputs, when a condition references a non-existent path, then the interpreter must raise a validation error and exit.

## 7. Execution Diagrams

stateDiagram-v2
[*] --> Initializer
Initializer --> Orchestrator
Orchestrator --> Skill_Loader: LOAD_SKILL
Skill_Loader --> Orchestrator
Orchestrator --> Stagnation_Detector: CALL_SUBAGENT
Stagnation_Detector --> Output_Guardrail: Stagnation=True
Stagnation_Detector --> SubAgent_Node: Stagnation=False
SubAgent_Node --> Circuit_Breaker_Watchdog
Circuit_Breaker_Watchdog --> Orchestrator: Safe
Circuit_Breaker_Watchdog --> [*]: FORCE_EXIT (interrupt)
Orchestrator --> Output_Guardrail: FINISH
Output_Guardrail --> [*]

````

```mermaid
stateDiagram-v2
    [*] --> initializer
    initializer --> orchestrator

    orchestrator --> skill_loader
    orchestrator --> stagnation_detector
    orchestrator --> output_guardrail

    stagnation_detector --> subagent
    stagnation_detector --> output_guardrail

    subagent --> circuit_breaker
    circuit_breaker --> orchestrator

    output_guardrail --> [*]
````

```mermaid
graph TD
    START([Start]) --> InputGuardrail

    InputGuardrail -->|Valid| GraphInitializer
    InputGuardrail -->|Invalid| FORCE_EXIT[Force Exit with Error]

    GraphInitializer --> SkillLoader

    SkillLoader -->|Tier1 Always| Orchestrator
    SkillLoader -->|Tier2 Lazy| Orchestrator

    Orchestrator --> StagnationDetector

    StagnationDetector -->|Stagnation Detected| FORCE_EXIT
    StagnationDetector -->|Normal| Router{Orchestrator Router}

    Router -->|SubAgent_X| SubAgentExecutor
    Router -->|FINISH| OutputGuardrail
    Router -->|FORCE_EXIT| FORCE_EXIT

    SubAgentExecutor -->|Summarized Result| Orchestrator
    SubAgentExecutor -->|Error| CircuitBreakerWatchdog

    CircuitBreakerWatchdog -->|Kill Flag Set| FORCE_EXIT
    CircuitBreakerWatchdog -->|OK| Orchestrator

    OutputGuardrail -->|Pass| END([End])
    OutputGuardrail -->|Fail| FORCE_EXIT

    FORCE_EXIT --> END
```

```mermaid
graph LR
    START([Start]) --> IG[InputGuardrail]

    IG -->|Valid| GI[GraphInitializer]
    IG -->|Invalid| FE[FORCE_EXIT]

    GI --> SL[SkillLoader<br/>Tier1 Always]
    SL --> CB[CircuitBreaker]

    CB -->|Kill Flag| FE
    CB -->|OK| ORCH[Orchestrator<br/>LLM]

    ORCH --> SD[StagnationDetector]
    SD -->|Stagnation| FE
    SD -->|Normal| ROUTER{Orchestrator<br/>Router}

    ROUTER -->|SubAgent| SA[SubAgentExecutor<br/>Isolated]
    ROUTER -->|FINISH| OG[OutputGuardrail]
    ROUTER -->|FORCE_EXIT| FE

    SA -->|Summarized| CB
    OG -->|Pass| END([End])
    OG -->|Fail| FE

    FE --> END
```

What this captures:

- The same compiled graph executor handles many workflows; dynamic configuration decides which branch executes.
- Level 1 frontmatter is always available, while Level 2 bodies are loaded lazily only when a branch needs them and Level 3 linked files are opened only when required.
- Subagents stay isolated and return summarized results before the orchestrator decides the next hop.
- The circuit breaker sits between subagent execution and the next orchestrator iteration so a thread can be stopped safely.
