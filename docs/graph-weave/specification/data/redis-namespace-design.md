## Redis Namespace Design

All runtime state lives in Redis.

## 1. Objective

- What: Define the Redis keyspace used by GraphWeave.
- Why: Keep workflow storage, runtime state, cache, and control flags cleanly separated.
- Who: Runtime and platform engineers.

## 2. Scope

- In scope: workflow JSON, version pointers, skill summaries, kill switches, active threads, compiled graphs, and checkpoints.
- Out of scope: business-domain source data and provider payloads.

## 3. Specification

- Workflow keys must be tenant-aware and versioned.
- Skill summaries and MCP schemas must use separate namespaces.
- Kill switches must support tenant, workflow, and thread blast radii.
- Active threads and compiled graphs must be independently addressable.

## 4. Technical Plan

- Use deterministic prefixes for each Redis concern.
- Keep checkpoint storage distinct from workflow definitions.
- Treat compiled graphs as cacheable artifacts with their own TTL strategy.

## 5. Tasks

- [ ] Define workflow pointer and version key conventions.
- [ ] Separate skill, kill-switch, and active-thread namespaces.
- [ ] Keep compiled graph cache and checkpoint storage distinct.

## 6. Verification

- Given a tenant-scoped workflow, when it is stored, then its version and pointer keys must be predictable.
- Given a kill switch, when it is triggered, then the matching scope must be stoppable without affecting other tenants.
- Given cached graphs and checkpoints exist, when they are read, then they must resolve from different namespaces.

| Namespace                                            | Purpose                                    |
| ---------------------------------------------------- | ------------------------------------------ |
| `workflow:{tenant_id}:{workflow_id}:v{version}`      | Stored workflow JSON                       |
| `workflow_pointer:{tenant_id}:{workflow_id}`         | Latest version pointer                     |
| `skills:tier1:{tenant_id}:{skill_id}`                | Tier 1 skill summaries                     |
| `skills:tier2:{tenant_id}:{skill_id}`                | Tier 2 MCP schemas                         |
| `ratelimit:{tenant_id}:tokens_per_min`               | Rate limiting bucket                       |
| `graphweave:circuit_breaker:{scope}:{id}:kill`       | Kill switch flags                          |
| `active_threads:{tenant_id}`                         | Active execution tracking                  |
| `compiled_graph:{tenant_id}:{workflow_id}:{version}` | Cached compiled graph                      |
| `langgraph_checkpoints:*`                            | LangGraph checkpoints managed by the saver |

Practical notes:

- Workflow pointers let consumers read the latest version without hardcoding a SemVer tag.
- Kill-switch keys are scoped by tenant, workflow, and thread so operators can stop execution at different blast radii.
- Active thread tracking is useful for audits, cancellation, and debugging long-running jobs.

Suggested hierarchy:

```text
graphweave:
├── workflow:
│   ├── {tenant_id}:{workflow_id}:latest -> "v2.1.0"
│   └── {tenant_id}:{workflow_id}:v{major}.{minor}.{patch} -> <JSON>
├── checkpoint:
│   └── {thread_id}:{step} -> <StateSnapshot>
├── mcp:
│   ├── registry:global -> {skill_id: endpoint_url}
│   └── {tenant_id}:{skill_id}:ratelimit -> {remaining: 59, reset: 1736942400}
├── circuit_breaker:
│   ├── tenant:{tenant_id}:kill -> bool
│   ├── workflow:{tenant_id}:{workflow_id}:kill -> bool
│   └── thread:{thread_id}:kill -> bool
├── active_threads:
│   └── {thread_id} -> {tenant_id, workflow_id, start_time, remaining_steps}
└── graph_cache:
    └── {tenant_id}:{workflow_id}:v{version} -> <CompiledGraph>
```

This layout keeps configuration, runtime state, cache, and emergency controls separate while still staying tenant-aware.
