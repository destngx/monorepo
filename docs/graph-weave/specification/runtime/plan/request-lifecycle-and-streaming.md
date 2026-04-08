# Request Lifecycle and Streaming

Purpose: record the two-request lifecycle so submission, run creation, checkpointing, and client streaming remain deterministic.

## Scope

- request intake and validation
- workflow fetch and run creation
- separate SSE status streaming
- checkpoint persistence and resume behavior
- completion cleanup

## References and context

- `[[../request-lifecycle]]` defines the canonical two-request behavior, including `POST /execute`, `GET /execute/{run_id}/status`, checkpointing, and active-thread cleanup. This reference anchors the plan to the client-visible runtime contract.
- `[[../README]]` frames runtime as execution flow, control loops, guardrails, and Redis-backed contracts. This context is included so the plan stays inside the runtime boundary and does not drift into architecture-level concerns.

## Decisions

- Validate before graph execution.
- Return a run id immediately from the submission request.
- Stream structured SSE events for request, node, tool, checkpoint, and completion milestones from the status request.
- Persist checkpoints during execution so interrupted runs can resume.
- Clear active thread state on completion.

## Definition of done

- The lifecycle from intake to completion is documented as submission plus status streaming.
- SSE and checkpoint responsibilities are explicit.
- The documented endpoints and event names remain stable for clients.

## Next links

- `[[universal-interpreter-and-skill-loading]]`
- `[[guardrails-circuit-breakers-and-stagnation]]`
