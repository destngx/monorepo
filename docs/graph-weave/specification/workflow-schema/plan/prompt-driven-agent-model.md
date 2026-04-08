# Prompt-Driven Agent Model

Purpose: record the separation between workflow structure, agent prompts, and dynamic skill loading.

## Scope

- system and user prompts
- agent autonomy
- dynamic skill loading
- branch evaluation on state, not skill availability

## References and context

- `[[../WORKFLOW_JSON_SPEC]]` explains that nodes carry prompts and agents decide which skills to load at runtime. This reference matters because it is the core design choice of the workflow graph model.
- `[[../MIGRATION_GUIDE]]` contrasts the old subagent-routing model with the new prompt-driven model. This reference is included because the plan must preserve the separation that migration introduces.

## Decisions

- Keep prompts on nodes, not in global routing rules.
- Let agents choose skills dynamically.
- Route by edge conditions on node output.
- Keep skill availability separate from workflow structure.
- Define agent-mode outputs as structured actions: goto node, call skill, or done.
- Keep agent state and execution history available to the prompt without exposing routing internals.

## Definition of done

- Prompt-driven autonomy is documented as a first-class design rule.
- Skill loading stays out of routing logic.
- The plan explains why deterministic edges and agent autonomy coexist inside the graph.
- The prompt contract defines the available skills, current state, and next-step expectations.

## Next links

- `[[schema-and-node-contracts]]`
- `[[schema-migration-and-compatibility]]`
