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

- 2026-04-08 00:00:00 — We clarified that SPEC work must proceed task-by-task with verification after each task, then completed GW-ARCH-001 and its verification.
- 2026-04-08 00:00:00 — We finished the architecture wave by aligning multi-tenant isolation and runtime request lifecycle docs, while removing PostgreSQL from task and verification files.
- 2026-04-08 00:00:00 — We resolved the thread_id/run_id model by making thread_id gateway-generated and documenting why two IDs can matter for reruns and recovery.
- 2026-04-08 00:00:00 — We cleaned SPEC docs to remove graph-driven wording, normalize level terminology, and eliminate PostgreSQL from all SPEC documents.
- 2026-04-08 00:00:00 — We replaced the skills registry feature with folder/frontmatter discovery plus Redis-backed lookup caching and explicit API-driven cache invalidation.
- 2026-04-08 00:00:00 — We removed skill-registry.json and the remaining registry framing from the skills docs, replacing it with discovery and cache language.
- 2026-04-08 00:00:00 — We added cache miss and refresh rules so external skill edits rebuild Redis lookup entries from folder/frontmatter source of truth.
- 2026-04-08 00:00:00 — We confirmed the SPEC phase docs are complete, but noted a few future implementation gaps like cache API shape and rerun semantics.
- 2026-04-09 00:00:00 — **Gap Analysis**: Identified that the initial 10 MOCK feature tasks lacked foundational infrastructure. Missing: app bootstrap, execution endpoint, mock storage, mock integrations, error handling, checkpoint storage.
- 2026-04-09 00:00:00 — **Phase Intent Clarified**: "MOCK" means a working prototype that boots, accepts requests, executes via mocked LangGraph, uses fake Redis, mocks MCP, returns responses, handles errors, and preserves state in memory.
- 2026-04-09 00:00:00 — **Foundation Tasks Created**: Added 6 new atomic tasks (GW-ARCH-103, GW-RUNTIME-103/104/105, GW-DATA-103, GW-SKILL-103) to provide the runnable foundation. Each task has 2-3 verification files (FUNC, DOC, SCHEMA types).
- 2026-04-09 00:00:00 — **Task Organization Decision**: All 16 MOCK tasks now live under component-local `tasks/MOCK/` folders with matching `verification/MOCK/` directories for better organization and phase isolation.
- 2026-04-09 00:00:00 — **MOCK Completion**: 16 atomic tasks + 37 verification files. All tasks follow the 74-line template with single, verifiable outputs. No scope overlap. Ready for junior developer implementation.
- 2026-04-09 00:00:00 — **MVP Direction Sketched**: Replace mocks with real implementations (real LangGraph, real Redis, real MCP). Keep happy-path focus; defer edge cases to FULL phase.
- 2026-04-09 00:00:00 — **FULL Direction Sketched**: Production hardening (tenant isolation, RBAC, rate limiting), security boundaries, observability, and comprehensive error recovery.
