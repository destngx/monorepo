# Architecture Living Spec

Use these docs for system context, platform boundaries, and multi-tenant layout.

See also: `[[plan/README]]`, `[[tasks/README]]`, `[[verification/README]]`.

## Component References

| Component           | File                      | Description                             | Scope                |
| ------------------- | ------------------------- | --------------------------------------- | -------------------- |
| Macro-Architecture  | `[[macro-architecture]]`  | System context, fixed stack, boundaries | Platform context     |
| Multi-Tenant Model  | `[[multi-tenant]]`        | Tenant, workflow, and thread isolation  | Runtime scoping      |
| System Architecture | `[[system-architecture]]` | Component interactions                  | Implementation guide |

## Phase Allocation

### [MOCK] Phase

- Request validation framework (pre-commit validator)
- Mock MCP servers with fixed responses
- Simple in-memory state (no Redis)

### [MVP] Phase

- Real FastAPI gateway with auth/rate limiting
- Real Redis with tenant-aware namespacing (single instance + prefixes)
- Real LangGraph interpreter with dynamic configuration
- Real MCP tool mediation
- Kill switches for individual threads (tenant-scoped)
- Checkpoint persistence for thread recovery

### [FULL] Phase

- Advanced multi-tenant features (isolation hardening)
- Separate Redis instances per tenant (optional upgrade from [MVP] single instance)
- Extended kill-switch patterns (tenant-wide, workflow-wide blast radius)
- Cross-tenant audit trails and billing hooks
- Advanced access control patterns
