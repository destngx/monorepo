---
name: repo_agent_memory
summary: Local repository copy of assistant memory entries from AGENTS.md
type: project
---

# Agent Memory (local copy)

## read_agents_md

Fact: Read AGENTS.md on repo load to extract agent definitions, responsibilities, and project-specific agent guidance.

Why: AGENTS.md is the authoritative guide for agent behavior and repo conventions referenced by CLAUDE.md.

How to apply: When starting work in this repo, consult this file and AGENTS.md; store only concise facts (roles, commands, conventions) and pointers, not full file dumps.

---

## monorepo_overview

Fact: Nx monorepo with TypeScript/Next.js, Go, and Python. Key apps:

- apps/wealth-management-engine (Go, Hexagonal) — active
- apps/wealth-management-dashboard (SvelteKit) — active
- apps/wealth-management (Next.js) — legacy
- apps/vnstock-server (Python) — active

How to apply: Prefer these paths for tasks and use the toolchain conventions (bun/bunx/uv) per AGENTS.md.

---

## agent_conventions (summary from AGENTS.md)

- Use bun / bunx / uv for scripts; do not use npm, yarn, or pnpm.
- Wealth engine follows Hexagonal layout: domain/, port/, service/, adapter/.
- Adapter naming: adapter/cache, adapter/config, adapter/db/..., adapter/market/....
- MCP server enabled on engine startup; run Go tests via Nx (bunx nx run wealth-management-engine:test).
- Testing: TDD + BDD; prefer real provider behavior tests for AI integrations.
- Security: .env.local is runtime-only; never commit secrets or credential files.

Refer to AGENTS.md for authoritative details.
