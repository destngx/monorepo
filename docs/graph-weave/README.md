# GraphWeave SDD Docs

This folder is organized by spec-driven development layers:

- `intent/` — why the system exists and which outcomes matter
- `specification/` — what the system must do, declaratively
- `plan/` — how the work is organized and sequenced
- `tasks/` — atomic implementation work items
- `verification/` — acceptance criteria, test cases, and evidence

Use `intent/` first, then `specification/`, then `plan/`, `tasks/`, and `verification/`.

## Questioning?

Main assumptions I made:

- GraphWeave is a real product/spec, not just a concept doc set? currently, it is only concept/spec, but the intent is to make it a real product spec and implementation guide.
- LangGraph + FastAPI + Redis + MCP are the intended core stack.
- Tiered skill loading, tenant/thread scoping, and 300s kill-switch TTL are canonical.
- The diagrams are meant to describe behavior, not exact implementation contracts.
  Questions to lock the spec down:

1. What is the primary goal of GraphWeave: product spec, internal architecture spec, or implementation guide? all of them, our aim is building the application and documenting it well enough that others can understand and build on it, also move to level: spec as sources of truth
2. Which docs are the source of truth: business-plan, technical/charts, or use-cases? not defined yet since all are just drafts and plan
3. Should every file in docs/graph-weave be converted to full SDD sections, or only selected files? all
4. Do you want a single canonical architecture, or are multiple alternatives acceptable in the docs? single final choice, but we can discuss alternatives in the docs as long as the final choice is clear
5. Is LangGraph mandatory, or just one possible runtime option? mandatory
6. Is FastAPI the final gateway choice, or should that remain abstract? final choice
7. Are Redis key names and TTLs fixed contractually, or just examples? examples, but we should have clear guidelines on how to choose them and what the implications are for different choices
8. What is the authoritative tenant model: tenant + thread, or tenant + workflow + thread? all of them, tenant + workflow + thread is the most complete model
9. Is tier1/tier2 skill loading a hard requirement, or an optimization pattern? hard requirement
10. What is the exact contract for “skills”: folder-based docs, runtime tools, or both? runtime load, and the tools for input could be either folder-based or MCP, but the key is that they are loaded dynamically at runtime rather than hardcoded in the graph
11. Should the docs specify concrete API endpoints, or stay endpoint-agnostic? concrete API endpoints, since the gateway is a key part of the system and we want to make sure the contract is clear for how clients interact with it
12. What event names are canonical for SSE / stream output? we do not decide yet, but we should have a clear naming convention and guidelines for how to choose event names based on the type of output (e.g. "workflow_started", "step_completed", "workflow_failed", etc.)
13. What is the intended maximum hop limit and stagnation threshold? 3 as default, but configurable per workflow, and we should have guidelines for how to choose these values based on the expected complexity and runtime of the workflow
14. Is the circuit breaker half-open state required, or optional? required, we want to make sure that workflows can recover gracefully from transient failures rather than just failing immediately when thresholds are exceeded, but the exact behavior in the half-open state can be configurable (e.g. allowing a certain percentage of requests through for testing)
15. Should output guardrails redact PII only, or also block policy topics? PII redaction is a must, but we also want to allow for blocking or redacting based on custom policies (e.g. blocking certain sensitive topics or keywords), so the guardrails should be flexible enough to support both use cases.
16. What are the required acceptance criteria for each use case? all ready described in the verification sections, but we should make sure that they are specific, measurable, and testable, and that they cover both functional and non-functional requirements (e.g. performance, reliability, security, etc.)
17. Do you want task checklists in the docs to map to actual implementation tickets? yes
18. Should verification be written as Gherkin, checklists, or both? checklists is good enough
19. Are diagrams authoritative spec, or illustrative only? we should have a clear distinction in the docs between diagrams that are meant to be authoritative specifications (e.g. architecture diagrams, sequence diagrams) and those that are meant to be illustrative or explanatory (e.g. high-level concept diagrams), but in general we want to aim for diagrams that are as close to the actual implementation as possible while still being clear and understandable.
20. Which parts are forbidden to change in implementation (external APIs, storage schema, naming, etc.)? the external APIs and the core architecture (e.g. the use of LangGraph, FastAPI, Redis, and MCP) should be considered contractually fixed, while internal implementation details (e.g. specific Redis key names, internal module structure, etc.) can be more flexible as long as they adhere to the overall specifications and do not break the external contracts or expected behavior of the system.
21. Should the docs include explicit non-functional requirements like latency, throughput, and reliability targets? yes, we should have clear non-functional requirements in the specification sections for each feature or component, and these should be measurable and testable in the verification sections (e.g. "Given a workflow with X steps, when it is executed, then it must complete within Y seconds with Z% success rate").
22. Do you want traceability IDs (FR-xxx, UC-xxx) added across the docs? yes
23. For use cases, do you want one “happy path” and multiple edge cases per scenario? yes
24. Should the docs include out-of-scope sections for each feature?they can be included in the specification sections to clarify what is not included in the scope of each feature or component, especially if there are common misconceptions or potential areas of confusion that we want to address upfront.
