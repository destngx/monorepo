# Graph Weave: Engineering & Agent Standards

This document defines the core engineering principles for the Graph Weave orchestration engine (Python).

## 🕸️ Architecture: LangGraph & MCP

The engine is built on **LangGraph** for stateful, multi-agent orchestration and **MCP** (Model Context Protocol) for tool integration.

### 🧩 Structural Components

- **Nodes**: Individual execution units (LLM call, Python tool, MCP tool call).
- **Edges**: Conditional logic routing the workflow based on State.
- **State**: Persistent Pydantic models tracking the context of a conversation/thread.

## 🛠️ Implementation Rules

### 📡 MCP & Routing

- Use the `mcp_router` to handle multiple MCP server integrations.
- Ensure tool calls are validated against the internal schema before being passed to LLMs via the AI Gateway.

### 💾 State & Persistence

- We use **Redis** for state checkpointing.
- Threads are identified by `thread_id`; always preserve state between node transitions.

### ⚖️ Circuit Breakers

- Implement Redis-based circuit breakers for high-latency external dependencies (Market Data, LLM Providers).

## 🧪 Testing & Workflow

- Use `pytest` for unit and integration tests.
- Mock AI Gateway responses to ensure deterministic testing of graph logic.
