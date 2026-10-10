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

### Git Worktrees (Required for Every Feature)

Multiple agent sessions work in this repo concurrently. Never do feature work in the primary checkout (`~/projects/monorepo`) or on a branch another session already has checked out - uncommitted changes from different features will collide.

- **Start every new feature in its own worktree**, branched from an up-to-date `main`:
  ```sh
  git fetch origin
  git worktree add -b <type>/<app>-<topic> ../monorepo-worktrees/<app>-<topic> origin/main
  cd ../monorepo-worktrees/<app>-<topic>
  pnpm install   # node_modules / .venv are per-worktree
  ```
- **Location**: always `../monorepo-worktrees/<app>-<topic>` (sibling of the repo, never nested inside it, so Nx and file watchers do not pick it up).
- **Check first**: run `git worktree list` and `git status` before editing. If the checkout has uncommitted changes unrelated to your task, stop and move to (or create) the correct worktree instead of editing alongside them.
- **One feature = one branch = one worktree.** Only touch files belonging to your feature.
- **Cleanup** after merge: `git worktree remove ../monorepo-worktrees/<app>-<topic>` and `git branch -d <branch>`.

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
