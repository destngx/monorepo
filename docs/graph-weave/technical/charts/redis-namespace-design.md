Redis Namespace Design

All runtime states are persisted in Redis.
```
  - Workflow Storage: workflow:{tenant_id}:{workflow_id}:{version} (String/JSON)

  - Latest Version Pointer: workflow_pointer:{tenant_id}:{workflow_id} (String)

  - MCP Registry: skills:tier1:{tenant_id}:{skill_id}, skills:tier2:{skill_id}:{tenant_id} (String/JSON)

  - Rate Limits: ratelimit:{tenant_id}:tokens_per_min (Sorted Set / Leaky Bucket)

  - Kill Switch: kill:{thread_id} (String Boolean, TTL 3600)

  - Active Threads: active_threads:{tenant_id} (Sorted Set, scored by timestamp)

  - Graph Cache: compiled_graph:{tenant_id}:{workflow_id}:{version} (Binary Pickle/Dill)

  - LangGraph Checkpoints: Managed internally by AsyncRedisSaver within the langgraph_checkpoints:* keyspace.

```
```
graphweave:
├── workflow:                          # Workflow storage
│   ├── {tenant_id}:{workflow_id}:latest -> "v2.1.0"
│   ├── {tenant_id}:{workflow_id}:v{major}.{minor}.{patch} -> <JSON>
│   └── {tenant_id}:{workflow_id}:v{major}.{minor}.{patch}:compiled -> <pickle>
├── checkpoint:                        # LangGraph checkpoints
│   └── {thread_id}:{step} -> <StateSnapshot>
├── mcp:                              # MCP registry
│   ├── registry:global -> {skill_id: endpoint_url}
│   └── {tenant_id}:{skill_id}:ratelimit -> {remaining: 59, reset: 1736942400}
├── circuit_breaker:                  # Kill switches
│   ├── tenant:{tenant_id}:kill -> bool (TTL 300s)
│   ├── workflow:{tenant_id}:{workflow_id}:kill -> bool
│   └── thread:{thread_id}:kill -> bool
├── active_threads:                   # Runtime tracking
│   └── {thread_id} -> {tenant_id, workflow_id, start_time, remaining_steps}
└── graph_cache:                      # Compiled graph cache
    └── {tenant_id}:{workflow_id}:v{version} -> <CompiledGraph>
```
