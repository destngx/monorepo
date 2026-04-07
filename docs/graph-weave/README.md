# GraphWeave SDD Docs

This folder is organized by spec-driven development layers:

- `intent/` — why the system exists and which outcomes matter
- `specification/` — what the system must do, declaratively
- `plan/` — how the work is organized and sequenced
- `tasks/` — atomic implementation work items
- `verification/` — acceptance criteria, test cases, and evidence

Use `intent/` first, then `specification/`, then `plan/`, `tasks/`, and `verification/`.

Source of truth rules:

- `intent/` defines the problem, outcomes, scope, and traceability IDs.
- `specification/` defines the canonical architecture, contracts, and runtime behavior.
- `plan/`, `tasks/`, and `verification/` refine delivery order and proof of correctness.
- External APIs and the core stack (`LangGraph`, `FastAPI`, `Redis`, `MCP`) are contractually fixed.
