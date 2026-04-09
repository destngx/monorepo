# GraphWeave Progress

## Phases

| Phase ID | Phase        | Status      | Goal                                                                        | Linked components                                                                                      |
| -------- | ------------ | ----------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| SPEC     | Spec lock    | completed   | Fully determine and stabilize the spec documents before coding starts       | `[[specification/README]]`, `[[intent/README]]`, `[[delta-changes]]`                                   |
| MOCK     | Mock         | in_progress | Build a working app with mocked external responses and mocked complex logic | `[[specification/workflow-schema/README]]`, `[[specification/architecture/README]]`                    |
| MVP      | MVP          | pending     | Implement the minimum executable workflow stack                             | `[[specification/runtime/README]]`, `[[specification/data/README]]`, `[[specification/skills/README]]` |
| FULL     | Full feature | pending     | Add guardrails, tenant isolation, and richer skill loading                  | `[[specification/runtime/README]]`, `[[specification/data/README]]`                                    |

## MOCK Phase Details

**Objective**: Create a working application that boots, accepts requests, executes workflows via mocked LangGraph, uses fake Redis, mocks MCP integrations, returns responses, handles errors, and preserves state in memory.

**Scope** (16 atomic tasks across 5 components):

| Component    | Tasks                                                                          | Coverage                                                                             |
| ------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------ |
| Architecture | GW-ARCH-101, GW-ARCH-102, GW-ARCH-103                                          | Colorized logging, dynamic OpenAPI docs, bootstrap core                              |
| Runtime      | GW-RUNTIME-101, GW-RUNTIME-102, GW-RUNTIME-103, GW-RUNTIME-104, GW-RUNTIME-105 | Stable run_id, new thread_id, execution endpoint, error handling, checkpoint storage |
| Data         | GW-DATA-101, GW-DATA-102, GW-DATA-103                                          | Versioned cache keys, latest fallback, mock Redis adapter                            |
| Skills       | GW-SKILL-101, GW-SKILL-102, GW-SKILL-103                                       | Cache invalidation API, cache miss rebuild, mock MCP integration                     |
| Workflow     | GW-WF-101, GW-WF-102                                                           | Single phase labels, multi-label support                                             |

**Verification**: 37 verification files (FUNC, DOC, and SCHEMA types) paired 2-3 per task.

**Task Organization**: All tasks live under phase folders (`tasks/MOCK/` with matching `verification/MOCK/` across all components).

**Acceptance**: All 16 tasks follow the 74-line atomic task template with single, verifiable outputs. No task has overlapping scope. Ready for junior developer implementation.

## MVP Phase Directions

**Objective**: Replace all mocks with real minimal implementations. Wire the core stack end-to-end: real LangGraph execution, real Redis, real MCP server integration.

**Scope Changes**:

- Replace mock Redis adapter with actual Redis backend (GW-DATA-103 → GW-DATA-201)
- Replace mock MCP integration with actual MCP protocol (GW-SKILL-103 → GW-SKILL-201)
- Replace mock error handling with production error paths (GW-RUNTIME-104 → GW-RUNTIME-201)
- Add real checkpoint persistence (GW-RUNTIME-105 → GW-RUNTIME-201)
- Keep happy-path focus; defer edge cases to FULL phase

**Success Criteria**: Happy-path workflow execution end-to-end with real external systems, all original MOCK verification files still passing.

## FULL Phase Directions

**Objective**: Production hardening, multi-tenant isolation, security guardrails, and richer behavioral control.

**Scope Additions**:

- Tenant isolation in Redis keys and execution context
- Role-based access control (RBAC) for skill and workflow management
- Rate limiting and quota management
- Comprehensive error recovery (retry, backoff, DLQ)
- Security boundary enforcement (MCP sandboxing, skill isolation)
- Observability (tracing, metrics, structured logging)

**Acceptance**: All FULL tasks reference both MOCK and MVP verification files to ensure no regressions.

## Rules

- keep phase status current
- link to component task files instead of duplicating task details
- update this file whenever a phase changes
- record notable debate outcomes and friction in `[[delta-changes]]`
- code work starts only after the Spec lock phase is complete
