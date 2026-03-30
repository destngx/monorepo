# Monorepo Knowledge Base

## Overview

Nx monorepo with mixed stacks:

- TypeScript/Next.js apps and shared libs
- Go backend engine for wealth-management
- Python backend for market data (`vnstock-server`)

Primary active backend migration target is `apps/wealth-management-engine` (Go, Hexagonal architecture).

## Current Focus Areas

| Area                      | Location                                            | Stack            | Status                 |
| ------------------------- | --------------------------------------------------- | ---------------- | ---------------------- |
| Wealth engine backend     | `apps/wealth-management-engine`                     | Go + Fiber + MCP | Active                 |
| Wealth dashboard frontend | `apps/wealth-management-dashboard`                  | SvelteKit        | Active                 |
| Legacy wealth app         | `apps/wealth-management` + `libs/wealth-management` | Next.js + React  | Reference/legacy track |
| Market data backend       | `apps/vnstock-server`                               | Python + vnstock | Active                 |

## Key Structure

```text
monorepo/
├── apps/
│   ├── wealth-management-engine/      # Go backend (ports/adapters)
│   ├── wealth-management-dashboard/   # SvelteKit frontend
│   ├── wealth-management/             # Legacy Next.js app
│   ├── vnstock-server/                # Python backend
│   ├── cloudinary-photos-app/
│   └── portfolio-landpage/
├── libs/
├── docs/
│   ├── tasks/
│   └── wealth-management/
├── go.work
├── go.work.sum
├── nx.json
└── package.json
```

## Documentation Conventions

- Sprint docs are folder-based under `docs/wealth-management/tasks/`:
  - `SPRINT_1/README.md`, `SPRINT_2/README.md`, etc.
  - Each task is a standalone file: `WM-xxx.md`.
- Global standards and standing conventions live in:
  - `docs/wealth-management/tasks/README.md` (source of truth for sprint process/conventions).

## Engineering Conventions (Must Follow)

### Runtime and Tooling

- Use `bun run` for root scripts.
- Use `bunx nx ...` for Nx tasks.
- Use `uv` for Python workflows (prefer via Nx scripts when available).
- Volta pins Node at major `22` in root `package.json`.

### Wealth Engine (Go)

- Architecture: strict Hexagonal (`domain/`, `port/`, `service/`, `adapter/`).
- `port` holds interfaces/contracts; `adapter` holds implementations.
- Domain naming:
  - Prefer business naming (`accounts`, not `accounts_sheet`).
  - Keep storage/provider details out of domain type names.
- Adapter naming:
  - `adapter/cache` (not `adapter/redis`)
  - `adapter/config` (not `adapter/env`)
  - `adapter/db/...` for database backends (Google Sheets is one backend)
  - `adapter/market/...` for market providers (`vnstock` is one implementation)
- MCP server is always enabled on engine startup.
- Go test target is executed via:
  - `bunx nx run wealth-management-engine:test`

### Test and Delivery Workflow

- Default workflow: TDD + BDD.
- Prefer real provider behavior tests for AI provider integrations where practical.
- Repo-local temp cache usage for Go tasks under `tmp/` (avoid root `/tmp` for project scripts).

### Security and Env

- `.env.local` is runtime input only.
- Never print or document secrets from `.env.local`.
- Never commit credential files.

## Useful Commands

```bash
# Nx
bunx nx show projects
bunx nx run-many -t lint
bunx nx run-many -t test

# Wealth engine
bunx nx serve wealth-management-engine
bunx nx run wealth-management-engine:test

# Wealth dashboard
bunx nx serve wealth-management-dashboard

# Python backend (if mapped in Nx, prefer Nx target)
uv run pytest apps/vnstock-server
```

## Anti-Patterns

- Do not use `npm`, `yarn`, or `pnpm` commands in this workspace.
- Do not bypass ports with cross-layer direct calls in Go engine.
- Do not patch workspace linking with ad-hoc TS path hacks.
- Do not swallow errors silently.
- Do not hardcode provider-specific logic into general domain contracts.
