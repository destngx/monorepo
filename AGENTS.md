# Monorepo Knowledge Base

## Overview

Nx monorepo with mixed stacks for wealth management, market data, and AI orchestration.

- **Goal**: Unified platform for financial intelligence and automated portfolio orchestration.
- **Stack**: Go (Backend/Gateway), SvelteKit (Dashboard), Python (Market Data/Agent Graphs), TypeScript (Libs).

## Knowledge Discovery (Progressive Disclosure)

Claude Code and other agentic harnesses use **Progressive Disclosure**. Read the specific documentation for the project area you are working on before starting tasks:

### 🏛️ Engineering Standards

- **Wealth Management**: [AGENTS.md](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/AGENTS.md) (Go Hexagonal / SvelteKit FSD)
- **AI Gateway**: [AGENTS.md](file:///Users/ez2/projects/personal/monorepo/docs/ai-gateway/AGENTS.md) (Clean Architecture / Provider Patterns)
- **Graph Weave**: [AGENTS.md](file:///Users/ez2/projects/personal/monorepo/docs/graph-weave/AGENTS.md) (LangGraph / MCP Orchestration)
- **Market Data**: [AGENTS.md](file:///Users/ez2/projects/personal/monorepo/docs/vnstock-server/AGENTS.md) (Python / Caching Policies)

### 🚀 Roadmap & Tasks

- **Active Tasks**: [Sprint README](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/README.md)
- **Product Vision**: [Product README](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/README.md)

## Engineering Conventions (Universal)

### Runtime and Tooling

- **Bun**: Use `bun run` and `bunx nx` for package management and task execution.
- **Python**: Use `uv` for Python environment management.
- **Workflow**: Default to **TDD + BDD**. Always verify changes with unit or integration tests.

### Security and Env

- `.env.local` is for runtime input ONLY. Never commit, print, or log secrets.
- Use `./tmp` for temporary files; avoid root `/tmp`.

## Anti-Patterns

- **No NPM/Yarn/PNPM**: Use `bun` exclusively in this workspace.
- **No Error Swallowing**: Always handle or wrap errors properly.
- **Check Docs First**: Review the corresponding `docs/*/AGENTS.md` before implementing architectural changes.
