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
- **Native Tools (OCR)**: [AGENTS.md](file:///Users/destnguyxn/projects/monorepo/docs/mac-ocr/AGENTS.md) (Swift / Vision Framework)

### 🚀 Roadmap & Tasks

- **Active Tasks**: [Sprint README](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/tasks/README.md)
- **Product Vision**: [Product README](file:///Users/ez2/projects/personal/monorepo/docs/wealth-management/README.md)

## Engineering Conventions (Universal)

### Runtime and Tooling

- **Node**: Use `pnpm run` and `pnpx nx` for package management and task execution.
- **Python**: Use `uv` for Python environment management. Always execute tests and Python scripts using the local virtual environment binaries directly (e.g., `.venv/bin/pytest` or `.venv/bin/python`) to ensure dependencies and configurations are correctly loaded and to avoid sandbox or path-resolution environment issues.
- **Workflow**: Default to **TDD + BDD**. Always verify changes with unit or integration tests.

### Security and Env

- `.env.local` is for runtime input ONLY. Never commit, print, or log secrets.
- Use `./tmp` for temporary files; avoid root `/tmp`.

## Anti-Patterns

- **No NPM/Yarn/bun**: Use `pnpm` exclusively in this workspace.
- **No Error Swallowing**: Always handle or wrap errors properly.
- **Check Docs First**: Review the corresponding `docs/*/AGENTS.md` before implementing architectural changes.
