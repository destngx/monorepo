# GraphWeave Progress

## Phases

| Phase | Status       | Goal                                                 | Linked components                                                                   |
| ----- | ------------ | ---------------------------------------------------- | ----------------------------------------------------------------------------------- |
| SPEC  | ✅ completed | Fully stabilize spec documents before coding         | `[[specification/README]]`, `[[intent/README]]`, `[[delta-changes.md]]`             |
| MOCK  | ✅ completed | Build working app with mocked responses and logic    | `[[specification/workflow-schema/README]]`, `[[specification/architecture/README]]` |
| MVP   | ⏳ pending   | Implement the minimal executable real workflow stack | `[[specification/runtime/README]]`, `[[specification/data/README]]`                 |
| FULL  | ⏳ pending   | Add guardrails, tenant isolation, RBAC, true scaling | `[[specification/runtime/README]]`, `[[specification/data/README]]`                 |

## MOCK Phase Status: ✅ COMPLETE

- **Scope**: Created 25 foundational and CRUD tasks across Architecture, Runtime, Data, Skills, and Workflow components.
- **Milestones**:
  - Implemented MockLangGraphExecutor, MockAIProvider, and 5 workflow CRUD endpoints.
  - Achieved 100% test coverage for MOCK scope (205 tests passing).
  - All endpoints handle basic validation, error standardization, and mock logic.

## MVP Phase Directions: ⏳ IN PROGRESS

**Objective**: Replace mocks with real minimum implementations (real LangGraph, real Redis, real MCP). LLM provider defaults to GitHub Copilot (`claude-3.5-sonnet`) using `GITHUB_TOKEN`.

**Task Refinements**: Split overly large tasks and then expanded the runtime hardening surface, resulting in 19 MVP tasks with documented dependencies and verification coverage.

**Task Index**:

| Component            | Tasks                                                                                                                                                                                                     | Status         |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| **Data Layer**       | GW-MVP-DATA-201A, GW-MVP-DATA-201B, GW-MVP-DATA-202, GW-MVP-DATA-203                                                                                                                                      | Specs complete |
| **Runtime Layer**    | GW-MVP-RUNTIME-201, GW-MVP-RUNTIME-202A, GW-MVP-RUNTIME-202B, GW-MVP-RUNTIME-202C, GW-MVP-RUNTIME-203, GW-MVP-RUNTIME-204, GW-MVP-RUNTIME-205, GW-MVP-RUNTIME-206, GW-MVP-RUNTIME-207, GW-MVP-RUNTIME-208 | Specs complete |
| **E2E Verification** | GW-MVP-E2E-001, GW-MVP-E2E-002, GW-MVP-E2E-003                                                                                                                                                            | Specs complete |

_See `[[specification/MVP-TASK-INDEX.md]]` for detailed dependencies and full task scope._

**Success Criteria**:

- DATA-201A passes tests with <50ms p99 latency.
- All 19 tasks pass assigned coverage (150+ targeted tests).
- 0 type errors, 0 lsp_diagnostics.
- MOCK phase tests (205 passing) must remain regression-free.

## FULL Phase Directions: ⏳ PENDING

**Objective**: Production hardening and behavioral control.

- **Scope**: Tenant isolation in Redis/execution, RBAC, rate limiting, quota management, comprehensive error recovery, MCP sandboxing, structured observability.

## Rules

- Keep phase status current.
- Link to component task files; don't duplicate task details here.
- Update this file whenever a phase changes.
- Record notable debate outcomes and friction in `[[delta-changes.md]]`.
- Code work only begins after spec locks.
