---
title: GraphWeave
summary: A FastAPI service that validates JSON workflow graphs, stores them per tenant in Redis, and runs them as LLM agent loops with MCP tools, plus a SvelteKit editor for building them.
year: 2026
status: active
stack: [Python, FastAPI, Redis, MCP, SvelteKit]
repo: https://github.com/destngx/monorepo/tree/main/apps/graph-weave
featured: true
order: 1
sfx: ドン
---

## The problem

Multi-step agent runs are easy to demo and hard to operate. I wanted a workflow to be data rather than code, so it can be validated before it runs, stored and versioned, resumed after a restart, and stopped if it misbehaves. Tool calls also need one controlled boundary instead of ad hoc code paths.

## What it does

- Stores workflow definitions as JSON, scoped by tenant, through CRUD endpoints under `/workflows`, with separate routers for nodes and schedules.
- Validates a workflow before it is stored: schema, referenced skills, graph structure, and limits.
- Runs workflows asynchronously via `POST /execute`, with status, cancel, and recover endpoints keyed by `run_id`.
- Supports agent, orchestrator, and CLI node types.
- Drafts a workflow from a natural-language intent with an LLM generator.
- Editor UI with a workflow registry, node list, inspector, and an execution runner that updates the graph as nodes run.

## How it's built

- Python backend with FastAPI. Services are initialized in a lifespan hook in `src/main.py`.
- Redis access goes through a namespaced client wrapper, and the spec requires tenant, workflow, and thread scoping for execution state.
- LLM calls go to the AI gateway project in this monorepo, configured with `AI_GATEWAY_URL`.
- Tools are exposed through an MCP router with bash, filesystem, and web tools. Allowed paths are set by environment variables.
- Frontend is SvelteKit with Svelte 5 runes and Tailwind v4, in `apps/graph-weave-ui`.
- I wrote the spec before the code. The `docs/graph-weave` tree has intent, specification, and plan layers, and the progress log shows the SPEC phase closed before MOCK started.

## Interesting bits

- Stagnation detection caps the number of node visits in a run, so a cycle cannot loop forever:

```python
def is_stagnated(self, max_hops: Optional[int] = None) -> bool:
    limit = max_hops if max_hops is not None else self.max_hops
    return self.total_visits >= limit
```

- The Redis circuit breaker has CLOSED, OPEN, and HALF_OPEN states with a failure threshold. The spec keeps it separate from stagnation detection: one handles kill state, the other handles repeated loops.
- The project started with a mock executor and mock AI provider. The MVP phase replaced them with real Redis, real execution, and real LLM calls.
- In April 2026 I changed direction and routed LLM calls through the AI gateway, so provider choice and fallback live in one service.

## What's next

- The FULL phase is still pending. Planned work includes tenant isolation in Redis and execution, RBAC, rate limiting, quota management, error recovery, MCP sandboxing, and structured observability.
- Kill switches at tenant, workflow, and thread scope, with half-open recovery, are specified but are FULL-phase work.
