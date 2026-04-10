# Specification

Purpose: define the canonical system contract in declarative form.

## Component map

| Component                    | Scope                                                                   |
| ---------------------------- | ----------------------------------------------------------------------- |
| `[[architecture/README]]`    | system context, macro boundaries, tenant isolation                      |
| `[[runtime/README]]`         | execution flow, control loops, guardrails, circuit breakers             |
| `[[data/README]]`            | Redis namespacing, state organization, cache contracts                  |
| `[[skills/README]]`          | skill packaging, loading policy, and three-level progressive disclosure |
| `[[workflow-schema/README]]` | prompt-driven workflow JSON, node/edge contracts, migration notes       |

## Boundaries

- Architecture docs define what exists and where the hard boundaries are.
- Runtime docs define what happens during execution.
- Data docs define storage and namespace contracts.
- Skills docs define how skills are discovered and loaded.
- Workflow schema docs define the executable workflow declaration.
- Requirements and features may carry inline phase labels like `[MOCK]`, `[MVP]`, and `[FULL]`; multiple labels are allowed when a requirement spans more than one phase.

## Environment Configuration Rules

**Reference Template**: `.env.example` at `apps/graph-weave/.env.example` (source of truth for available keys)

**Runtime Source**: `.env.local` at `apps/graph-weave/.env.local` (never read directly; use `.env.example` as reference only)

### Available Environment Keys

| Key                        | Purpose                                       | Used By                                                | Phase |
| -------------------------- | --------------------------------------------- | ------------------------------------------------------ | ----- |
| `UPSTASH_REDIS_REST_URL`   | Upstash Redis instance endpoint               | All data layer tasks, all runtime tasks, all E2E tests | [MVP] |
| `UPSTASH_REDIS_REST_TOKEN` | Bearer token for Upstash API authentication   | All data layer tasks, all runtime tasks, all E2E tests | [MVP] |
| `GITHUB_TOKEN`             | GitHub API token for skill/workflow discovery | Runtime tasks (skill loading), E2E tests               | [MVP] |

### Tasks Referencing Each Key

**`UPSTASH_REDIS_REST_URL` and `UPSTASH_REDIS_REST_TOKEN`**:

- `[[runtime/tasks/MVP/GW-MVP-RUNTIME-201.md]]` (Status Lifecycle: Redis state transitions)
- `[[runtime/tasks/MVP/GW-MVP-RUNTIME-202.md]]` (LangGraph Executor: checkpoint retrieval)
- `[[runtime/tasks/MVP/GW-MVP-RUNTIME-203.md]]` (Event Emitter: event persistence)
- `[[data/tasks/MVP/GW-MVP-DATA-201.md]]` (Real Redis Integration: Upstash REST client implementation)
- `[[data/tasks/MVP/GW-MVP-DATA-202.md]]` (Checkpoint Persistence: save/restore from Redis)
- `[[data/tasks/MVP/GW-MVP-DATA-203.md]]` (Active Thread Lifecycle: thread tracking in Redis)
- `[[verification/MVP/GW-MVP-E2E-001.md]]` (Workflow Submission → Execution: Redis state validation)
- `[[verification/MVP/GW-MVP-E2E-002.md]]` (Agent + MCP Tools: checkpoint store for recovery)
- `[[verification/MVP/GW-MVP-E2E-003.md]]` (Checkpoint Recovery & Cancellation: Redis cleanup validation)

**`GITHUB_TOKEN`**:

- `[[runtime/tasks/MVP/GW-MVP-RUNTIME-202.md]]` (LangGraph Executor: GitHub skill loading)
- `[[verification/MVP/GW-MVP-E2E-002.md]]` (Agent + MCP Tools: GitHub-based skill verification)

### Configuration Loading Strategy

**Phase**: [MVP]

**Method**: Environment variable injection via configuration loader in `src/config.py`

**Pattern**:

```python
# src/config.py (pseudo-code)
import os

class Config:
    UPSTASH_REDIS_REST_URL = os.getenv("UPSTASH_REDIS_REST_URL")
    UPSTASH_REDIS_REST_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN")
    GITHUB_TOKEN = os.getenv("GITHUB_TOKEN", "")  # Optional, defaults to empty

    @staticmethod
    def validate():
        """Ensure required keys are present at startup"""
        required = ["UPSTASH_REDIS_REST_URL", "UPSTASH_REDIS_REST_TOKEN"]
        for key in required:
            if not getattr(Config, key.lower(), None):
                raise ValueError(f"Missing required env var: {key}")
```

**Validation Responsibility**: Each task must validate required keys at test initialization or runtime startup.

**Testing Strategy**: All unit/integration/E2E tests must load config and validate before execution.

## Review rule

If a decision changes runtime behavior, storage shape, or tenant isolation, update the relevant spec doc before implementation.

## Change memory

Use `[[../delta-changes]]` for incremental decisions, debate outcomes, and friction notes that should not be compressed away.
