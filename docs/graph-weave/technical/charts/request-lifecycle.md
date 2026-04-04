```mermaid
sequenceDiagram
    participant C as Client
    participant API as FastAPI Gateway
    participant R as Redis
    participant V as Pre-Commit Validator
    participant G as Universal Interpreter
    participant M as MCP Servers
    participant S as SSE Streamer

    C->>API: POST /execute {workflow_id, input}
    API->>R: Check quota + fetch workflow
    API->>V: Validate workflow JSON
    V-->>API: Validation passed
    API->>G: Start graph with config + state

    loop Each graph step
        G->>R: Load / save checkpoint
        G->>M: Call tool if needed
        M-->>G: Summarized tool result
        G->>S: Emit node/token/tool events
        S-->>C: SSE stream
    end

    G-->>API: Final state
    API->>R: Clear active thread
    API-->>C: Complete response
```

Key runtime details:

- The graph streams granular events such as `node_start`, `tool_start`, `tool_result`, `token`, `node_end`, and `complete`.
- Checkpoints are written during execution so a thread can resume after interruption.
- The active thread key is cleared at the end of the run, which keeps concurrency and kill-switch handling predictable.
