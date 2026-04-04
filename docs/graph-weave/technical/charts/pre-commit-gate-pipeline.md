```mermaid
flowchart LR
    A[Workflow JSON submitted] --> B[Pydantic schema validation]
    B -->|invalid| X1[Reject: schema error]
    B -->|valid| C[MCP / skill integrity check]
    C -->|missing refs| X2[Reject: missing skill]
    C -->|ok| D[NetworkX DAG validation]
    D -->|cycles or dead nodes| X3[Reject: graph error]
    D -->|ok| E[SemVer + limit checks]
    E -->|invalid| X4[Reject: version or limit error]
    E -->|ok| F[Store workflow + cache graph]
    F --> G[Return workflow ID + version]
```

Validation notes:

- Schema errors fail fast before any storage occurs.
- MCP or skill references must exist before a workflow is accepted.
- Cycles and dead nodes are rejected when they indicate bypasses, while the orchestrator’s intentional runtime loop remains a separate concern.
- SemVer and limit checks prevent incompatible versions and unsafe resource settings from entering the runtime.
