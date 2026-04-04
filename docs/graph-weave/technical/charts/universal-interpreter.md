```mermaid
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
```

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
```

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

- The same compiled graph handles many workflows; dynamic configuration decides which branch executes.
- Tier 1 summaries are always available, while Tier 2 schemas are loaded lazily only when a branch needs them.
- Subagents stay isolated and return summarized results before the orchestrator decides the next hop.
- The circuit breaker sits between subagent execution and the next orchestrator iteration so a thread can be stopped safely.
