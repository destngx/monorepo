# GraphWeave Progress

## Phases

| Phase ID | Phase        | Status      | Goal                                                                  | Linked components                                                                                      |
| -------- | ------------ | ----------- | --------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| SPEC     | Spec lock    | in_progress | Fully determine and stabilize the spec documents before coding starts | `[[specification/README]]`, `[[intent/README]]`, `[[delta-changes]]`                                   |
| MOCK     | Mock         | pending     | Validate the prompt-driven architecture and core boundaries           | `[[specification/workflow-schema/README]]`, `[[specification/architecture/README]]`                    |
| MVP      | MVP          | pending     | Implement the minimum executable workflow stack                       | `[[specification/runtime/README]]`, `[[specification/data/README]]`, `[[specification/skills/README]]` |
| FULL     | Full feature | pending     | Add guardrails, tenant isolation, and richer skill loading            | `[[specification/runtime/README]]`, `[[specification/data/README]]`                                    |

## Rules

- keep phase status current
- link to component task files instead of duplicating task details
- update this file whenever a phase changes
- record notable debate outcomes and friction in `[[delta-changes]]`
- code work starts only after the Spec lock phase is complete
