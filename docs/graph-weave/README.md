# GraphWeave Living Spec

Project ID: `GW`

Use this short id to prefix GraphWeave-scoped tasks and documents across the monorepo so they stay isolated and easy to search.

This repo uses a layered living-spec structure. Each layer answers one question:

| Layer        | Question                                      | Source                                                                                                                                             |
| ------------ | --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Vision       | Why does this exist?                          | `[[intent/README]]`                                                                                                                                |
| Architecture | What are the system boundaries?               | `[[specification/architecture/README]]`                                                                                                            |
| Components   | What are the major subsystems?                | `[[specification/runtime/README]]`, `[[specification/data/README]]`, `[[specification/skills/README]]`, `[[specification/workflow-schema/README]]` |
| Delivery     | How do we sequence work and record decisions? | Component-local `[[plan/README]]` and `[[tasks/README]]` folders under each component                                                              |
| Proof        | How do we know it works?                      | Component-local `[[verification/README]]` folders under each component                                                                             |

## Memory layer

| Layer          | Question                                           | Source              |
| -------------- | -------------------------------------------------- | ------------------- |
| Delta history  | What changed, why, and what friction did we learn? | `[[delta-changes]]` |
| Phase progress | What phase are we in right now?                    | `[[progress]]`      |

## Working rules

1. Start in `intent/` to confirm the problem, scope, and success definition.
2. Use `specification/` to define the canonical behavior, boundaries, and contracts.
3. Use component-local `plan/` first for sequencing, trade-offs, blockers, and decision log entries.
4. Then use component-local `tasks/` for atomic, independently completable work items.
5. Then use component-local `verification/` for DoD, acceptance criteria, regression coverage, and evidence.
6. Use `delta-changes.md` for incremental history, debate outcomes, and friction.
7. Use `progress.md` for phase status and high-level delivery tracking.

## Naming rules

- Use `GW` as the short id for GraphWeave artifacts.
- Prefer prefixes like `GW-ARCH-001`, `GW-RUNTIME-001`, `GW-DATA-001`, `GW-SKILL-001`, and `GW-WF-001`.
- Keep the prefix stable across tasks, plan docs, and other GraphWeave documents.

## Default delivery flow

Plan first → tasks next → verification last.

This order is iterative, but the default loop is:

1. decide the plan
2. split it into atomic tasks
3. define verification for each task

## Boundary rules

- `intent/` stays product-level: no implementation detail, no code.
- `specification/` stays declarative: define contracts, not steps.
- `plan/`, `tasks/`, and `verification/` live inside the component they describe.
- `plan/` captures decisions that affect delivery order or architecture trade-offs.
- `tasks/` should be one acceptance criterion per item.
- `verification/` should prove behavior with Given/When/Then or checklists.

## Phase intent

- **MOCK**: A working application with mocked responses for external systems and complicated logic.
- **MVP**: Real minimal execution path with the core stack wired end-to-end.
- **FULL**: Production hardening, guardrails, multi-tenant correctness, and richer loading behavior.

## Tier rules

- **Auto-done**: formatting, docs cleanup, verification updates, routine evidence capture.
- **Human-in-loop**: security boundaries, tenant isolation, workflow semantics, architecture trade-offs.
- **Forbidden**: secrets, production deploys, irreversible data changes.

## Fixed platform contract

The external API and core stack are contractually fixed: `LangGraph`, `FastAPI`, `Redis`, and `MCP`.

## Living-spec goal

Keep the docs bidirectional: implementation updates the spec, and the spec constrains implementation.
