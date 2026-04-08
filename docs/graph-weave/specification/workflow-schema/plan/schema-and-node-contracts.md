# Schema and Node Contracts

Purpose: record the declarative workflow graph shape so node behavior, edge routing, and guardrails remain explicit.

## Scope

- top-level JSON fields
- node types and configs
- edge conditions and labels
- guardrails and output mappings

## References and context

- `[[../WORKFLOW_JSON_SPEC]]` is the authoritative schema for nodes, edges, guardrails, and limits. This reference anchors the plan to the contract that workflow authors must follow.
- `[[../README]]` states that this folder is the local entrypoint for the prompt-driven workflow schema. This context matters because the plan is about schema shape, not implementation code.

## Decisions

- Keep nodes and edges as the source of truth.
- Attach guardrails to the node where the risk lives.
- Use output mappings for deterministic exits.
- Keep edge conditions deterministic and JSONPath-driven.
- Keep schema versioning explicit so compatible workflows can be validated predictably.

## Definition of done

- The graph shape is documented in plan form.
- Node and edge contracts are easy to trace back to the authoritative spec.
- The plan makes guardrail placement and output mapping visible.

## Next links

- `[[prompt-driven-agent-model]]`
- `[[schema-migration-and-compatibility]]`
