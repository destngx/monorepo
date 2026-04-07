# Specification

Purpose: define the canonical system contract in declarative form.

## Component map

| Component                    | Scope                                                             |
| ---------------------------- | ----------------------------------------------------------------- |
| `[[architecture/README]]`    | system context, macro boundaries, tenant isolation                |
| `[[runtime/README]]`         | execution flow, control loops, guardrails, circuit breakers       |
| `[[data/README]]`            | Redis namespaces, state organization, cache contracts             |
| `[[skills/README]]`          | skill packaging, loading policy, Tier 1/Tier 2 rules              |
| `[[workflow-schema/README]]` | prompt-driven workflow JSON, node/edge contracts, migration notes |

## Boundaries

- Architecture docs define what exists and where the hard boundaries are.
- Runtime docs define what happens during execution.
- Data docs define storage and namespace contracts.
- Skills docs define how skills are discovered and loaded.
- Workflow schema docs define the executable workflow declaration.

## Review rule

If a decision changes runtime behavior, storage shape, or tenant isolation, update the relevant spec doc before implementation.

## Change memory

Use `[[../delta-changes]]` for incremental decisions, debate outcomes, and friction notes that should not be compressed away.
