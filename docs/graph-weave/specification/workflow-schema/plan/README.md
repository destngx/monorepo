# Plan

Purpose: capture workflow-schema sequencing, migration trade-offs, and node/edge contract decisions.

## Use for

- JSON schema evolution
- migration steps from older workflow models
- node, edge, and prompt contract trade-offs

## Rules

- keep entries short and decision-focused
- record compatibility impact for schema changes
- update before changing workflow declaration shape or migration behavior

## See also

- `[[../README]]`
- `[[../tasks/README]]`
- `[[../verification/README]]`

## Plan files

- `[[implementation-node-registry]]` — **Implementation plan**: step-by-step execution for node registry changes
- `[[decomposed-generation-and-node-registry]]` — **Architecture plan**: node registry, decomposed generation, and reusable node library
- `[[schema-and-node-contracts]]` — node types, edge rules, guardrails, and output mappings
- `[[prompt-driven-agent-model]]` — prompts, autonomy, and skill-loading separation
- `[[schema-migration-and-compatibility]]` — old-model migration and backward-compatibility intent
- `[[api-organization]]` — [MOCK] Swagger UI semantic tag grouping design and endpoints organization
