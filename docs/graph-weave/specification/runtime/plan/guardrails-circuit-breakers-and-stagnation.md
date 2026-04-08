# Guardrails, Circuit Breakers, and Stagnation

Purpose: record the runtime safety controls that stop unsafe, stalled, or ambiguous execution.

## Scope

- kill-switch lifecycle
- watchdog interruption
- half-open recovery
- stagnation detection and meta-message injection

## References and context

- `[[../circuit-breaker]]` defines the kill-switch lifecycle, half-open recovery, and fail-closed behavior. This reference is included because the plan must preserve safe interruption semantics.
- `[[../stagnation-detection-logic]]` defines repeated-routing detection and the safe exit path. This reference matters because runaway loop prevention is part of the runtime safety story.

## Decisions

- Interrupt at the watchdog boundary.
- Support tenant, workflow, and thread-scoped kill switches.
- Fail closed when breaker state is ambiguous.
- Detect stagnation with a sliding window and route to the guardrail path.
- Keep stagnation detection and kill-switch lifecycle separate, but let both feed the same safe-exit path.

## Definition of done

- Safety controls are documented with clear blast-radius rules.
- Stagnation and breaker behavior are separated but linked.
- The plan explains how the runtime exits safely without corrupting state.

## Next links

- `[[request-lifecycle-and-streaming]]`
- `[[universal-interpreter-and-skill-loading]]`
