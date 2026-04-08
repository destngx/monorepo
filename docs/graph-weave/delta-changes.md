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
