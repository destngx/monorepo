# GraphWeave (archived)

Archived on 2026-10-22. The code and the living spec were removed from `main`.

|                             |                                                                                                                        |
| --------------------------- | ---------------------------------------------------------------------------------------------------------------------- |
| Last commit with GraphWeave | `61c58ea42f045c2ee3eac145d6e9d26cbee6a2f3` (2026-10-20)                                                                |
| Code at that commit         | [apps/graph-weave](https://github.com/destngx/monorepo/tree/61c58ea42f045c2ee3eac145d6e9d26cbee6a2f3/apps/graph-weave) |
| Spec at that commit         | [docs/graph-weave](https://github.com/destngx/monorepo/tree/61c58ea42f045c2ee3eac145d6e9d26cbee6a2f3/docs/graph-weave) |

## What it was

A FastAPI service that stored JSON workflow graphs per tenant in Redis, validated them before storage, and ran them as LLM agent loops with MCP tools. LLM calls went through `ai-gateway`. It had a SvelteKit editor, a natural-language workflow generator, hop limits, and a Redis circuit breaker. The SPEC, MOCK, and MVP phases were complete; the FULL phase (tenant isolation, RBAC, quotas, sandboxing) was never started.

## Why it was archived

- Its only real workload was the inbox ingestion pipeline in the Obsidian vault. No app in this repo called it.
- n8n, Dify, Flowise, and Langflow already store, validate, and run workflows as data, and n8n generates workflows from a prompt.
- Its tiered skill loading and isolated subagents are what Claude Code skills and subagents already do.
- The FULL phase only pays off with real tenants, and there were none.

## What replaced it

The vault's inbox ingestion runs as an agent skill (`.agents/skills/inbox-ingest`) called headless with `claude -p --json-schema`. Bash scripts do the deterministic parts (claiming notes, OCR with `mac-ocr`, validation, writing files). The LLM only does extraction.

## Restore

To look at the code locally:

```sh
git show 61c58ea4:apps/graph-weave/src/main.py
git worktree add ../monorepo-worktrees/graph-weave-archive-view 61c58ea4
```

To bring it back onto a branch:

```sh
git checkout 61c58ea4 -- apps/graph-weave docs/graph-weave
```

It needs Python with `uv`, a Redis instance, and `ai-gateway` running.
