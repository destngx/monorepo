# Runtime Execution Flow

Purpose: record how a request moves through validation, execution, state, and streaming so the runtime remains deterministic.

## Scope

- request handling
- validation before execution
- Redis-backed state and checkpoints
- LangGraph execution flow
- SSE event contract and observability hooks

## References and context

- `[[../system-architecture]]` describes the end-to-end runtime path and the expected SSE event types. This reference matters because the plan must preserve client-facing request and stream behavior.
- `[[../macro-architecture]]` clarifies the platform boundary around validation and interpreter execution. This reference is included so the execution flow stays aligned with the documented layer split.

## Decisions

- Validate workflow requests before execution begins.
- Route execution through the interpreter after gateway checks.
- Persist runtime state and checkpoints in Redis.
- Emit request, node, tool, checkpoint, and completion events in a stable SSE convention.

## Definition of done

- The request-to-completion path is documented as a single sequence.
- State and checkpoint responsibilities are assigned to Redis.
- SSE event names and ordering are described in the plan.

## Next links

- `[[platform-boundary-and-fixed-stack]]`
- `[[tenant-isolation-and-scoping]]`
