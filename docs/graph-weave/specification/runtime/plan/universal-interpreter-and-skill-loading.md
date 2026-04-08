# Universal Interpreter and Skill Loading

Purpose: record the reusable graph executor shape, node order, and lazy skill-loading behavior for the runtime.

## Scope

- compiled graph executor behavior
- initializer and orchestrator order
- lazy Level 2 loading
- on-demand Level 3 linked-file loading
- subagent isolation
- output guardrail and safe exit

## References and context

- `[[../universal-interpreter]]` describes the canonical graph traversal sequence and the requirement that one compiled graph executor handle multiple workflows. This reference matters because the plan must preserve deterministic execution behavior.
- `[[../../skills/plan/skill-loading-and-packaging]]` will describe how skill loading is planned from the skills side; this cross-reference exists because the runtime depends on that loading contract while keeping responsibilities separate.

## Decisions

- Keep one compiled graph executor for multiple workflows.
- Load Level 1 frontmatter before routing and Level 2 only on demand.
- Open Level 3 linked files only when needed by the selected skill.
- Keep subagent work isolated and summarized.
- Make safe-exit paths explicit through stagnation, breaker, and output-guardrail routes.
- Treat the interpreter as a consumer of pre-loaded skills, not a skill fetcher.
- Define interpreter I/O as compiled graph + state + context + skills in, result + events + deferred requests out.

## Definition of done

- The graph traversal sequence is documented as a reusable runtime pattern.
- Lazy loading and safe-exit rules are visible in the plan.
- The runtime path can be read without opening implementation code.

## Next links

- `[[request-lifecycle-and-streaming]]`
- `[[guardrails-circuit-breakers-and-stagnation]]`
