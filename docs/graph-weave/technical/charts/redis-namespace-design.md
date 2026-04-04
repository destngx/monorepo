## Redis Namespace Design

All runtime state lives in Redis.

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
