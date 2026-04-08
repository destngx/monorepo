# Platform Boundary and Fixed Stack

Purpose: record the immutable architecture boundary so implementation stays aligned with the contractually fixed platform.

## Scope

- FastAPI gateway
- LangGraph execution boundary
- Redis-backed runtime state
- MCP as the external tool boundary
- external API contracts for client integration

## References and context

- `[[../system-architecture]]` defines the fixed stack and the external API contract. This reference anchors the plan to the canonical end-to-end platform shape so delivery cannot drift into an alternate stack.
- `[[../macro-architecture]]` defines the request, validation, runtime, and external tool layers. This reference is included because the plan uses those layers to keep the gateway thin and the interpreter isolated.

## Decisions

- Keep the API gateway stateless and narrow.
- Preserve FastAPI, LangGraph, Redis, and MCP as the fixed core stack.
- Treat the interpreter as the single workflow runtime.
- Keep PostgreSQL for registry and audit support, not runtime state.

## Definition of done

- The fixed stack is documented in one place and referenced from the component plan.
- The boundary between gateway, runtime, and external tools is explicit.
- No plan item introduces a new core stack dependency without updating the spec first.

## Next links

- `[[runtime-execution-flow]]`
- `[[tenant-isolation-and-scoping]]`
