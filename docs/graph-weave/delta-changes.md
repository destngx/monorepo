# GraphWeave Delta Changes

Purpose: record incremental updates, debate outcomes, choices, friction, and decision history without compacting away context.
This file is the incremental memory for GraphWeave. It preserves choices and context better than compaction alone.

## Use this file for

- debate results and why a choice was made
- progress snapshots (done / ongoing / blocked)
- friction found during implementation
- decisions that would otherwise be debated again
- links to the exact component docs or task files changed

## Content

## Conversation Log

- 2026-04-10 00:00:00 — **MVP TODO Expansion**: Converted the remaining runtime TODO clusters in `src/main.py` into formal MVP task/verification docs. Added GW-MVP-RUNTIME-204 through GW-MVP-RUNTIME-208 for execution hardening and workflow CRUD semantics, then expanded the MVP task index from 14 to 19 tasks so the registry matches the new scope.

- 2026-04-08 — **Architecture & SPEC**: Completed architecture alignment, removing PostgreSQL from SPEC docs. Resolved `thread_id`/`run_id` model (gateway generates `thread_id`). Replaced JSON skills registry with folder/frontmatter discovery and Redis-backed caching.

- 2026-04-09 — **MOCK Phase Setup**: Clarified MOCK phase intent (prototype with mocked LangGraph, fake Redis, MCP). Created 25 foundational and CRUD tasks under component-local `tasks/MOCK/` to provide a runnable foundation.
- 2026-04-09 — **Validation Implementation**: Implemented Pydantic request/response validation and unified error formats. Fixed pre-existing issues by making `input` explicitly required and using custom `RequestValidationError` exception handlers for 400/422 consistency (e.g., explicit endpoint validation for `tenant_id`).
- 2026-04-09 — **API Contract Debates**: Removed client-provided `run_id` from ExecuteRequest; it is now server-generated and returned in ExecuteResponse to ensure the server owns execution identity.
- 2026-04-09 — **Workflow CRUD**: Implemented 5 mock endpoints (POST, GET, GET list, PUT, DELETE) using in-memory `MockWorkflowStore`. Enforced test isolation via `conftest.py`. Added Swagger API tags to organize endpoints into Execution, Skills, and Workflows.
- 2026-04-09 — **Mock Execution Engine**: Built MockAIProvider (keyword-based prompt routing) and MockLangGraphExecutor (node traversal, event logging) to support POST `/execute` and GET `/execute/{run_id}/status`.
- 2026-04-09 — **Tech Debt**: Resolved Pydantic v2 and FastAPI lifespan context manager deprecation warnings.
- 2026-04-09 — **MOCK Phase Completion**: Achieved 100% test coverage with mock functionality operational.
- 2026-04-09 — **MVP Specification Locked**: Analyzed MOCK gaps and locked 7 MVP decisions into spec docs:
  1. Return 404 for missing runs (not 200).
  2. Execution status lifecycle includes queued/validating states (202 Accepted).
  3. Workflow pre-creation is required before execution.
  4. Standardized 18 event types.
  5. JSON polling protocol (SSE deferred to FULL).
  6. Single Redis instance with tenant prefixing.
  7. No API version prefixes yet.

- 2026-04-10 — **Final Confirmation**: Re-ran both GraphWeave Nx test targets successfully (208 passing tests) to prove stability after code cleanups.
