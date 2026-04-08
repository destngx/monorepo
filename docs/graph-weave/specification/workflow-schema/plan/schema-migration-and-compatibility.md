# Schema Migration and Compatibility

Purpose: record the transition from the old subagent-routing model to the explicit prompt-driven workflow model.

## Scope

- old-to-new model mapping
- migration steps
- compatibility expectations
- validation and visualization intent

## References and context

- `[[../MIGRATION_GUIDE]]` explains why the old model was fragile and how the new model separates structure from content. This reference matters because the plan must preserve the reason for the migration, not just its mechanics.
- `[[../README]]` marks this folder as the local entrypoint for workflow schema docs. This context is included so migration remains part of the workflow-schema component, not a runtime side note.

## Decisions

- Treat explicit nodes and edges as the new baseline graph shape.
- Keep skills out of routing decisions.
- Preserve compatibility by validating and visualizing before execution.
- Keep migration notes traceable to the authoritative schema docs.

## Definition of done

- The old model’s problems and the new model’s benefits are documented together.
- Migration steps are easy to follow from the plan files.
- The plan makes compatibility implications visible before implementation changes.

## Next links

- `[[schema-and-node-contracts]]`
- `[[prompt-driven-agent-model]]`
