```mermaid
C4Context
    title GraphWeave system context

    Person(user, "End User", "Submits workflow requests")

    System_Boundary(graphweave, "GraphWeave Platform") {
        Container(api, "FastAPI Gateway", "Python/FastAPI", "Auth, rate limiting, request streaming")
        Container(validator, "Pre-Commit Validator", "Python/Pydantic", "Validates workflow JSON before storage")
        Container(interpreter, "Universal Interpreter", "LangGraph", "Single compiled graph with dynamic configuration")
        Container(redis, "Redis Cluster", "Redis 7.2+", "State, checkpoints, cache, kill switches")
        ContainerDb(postgres, "PostgreSQL", "Optional", "Admin and audit storage")
    }

    System_Ext(mcp_servers, "MCP Servers", "External tool providers")

    Rel(user, api, "HTTP POST / SSE", "JSON")
    Rel(api, validator, "Validates", "workflow JSON")
    Rel(api, interpreter, "Creates thread", "thread_id")
    Rel(interpreter, redis, "Reads / writes", "state and checkpoints")
    Rel(interpreter, mcp_servers, "Calls tools", "MCP protocol")
    Rel(validator, redis, "Stores validated workflows", "definitions")
```
