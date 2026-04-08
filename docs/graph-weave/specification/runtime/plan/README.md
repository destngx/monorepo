# Plan

Purpose: capture runtime sequencing, trade-offs, and control-loop decisions.

## Use for

- execution flow decisions
- guardrail and control-loop trade-offs
- recovery and failure policy decisions

## Rules

- keep entries short and decision-focused
- record why a control exists, not just what it does
- update before changing execution semantics or retry behavior

## See also

- `[[../README]]`
- `[[../tasks/README]]`
- `[[../verification/README]]`

## Plan files

- `[[request-lifecycle-and-streaming]]` — request validation, SSE, checkpoints, and completion flow
- `[[universal-interpreter-and-skill-loading]]` — compiled graph behavior, lazy loading, and safe exit paths
- `[[guardrails-circuit-breakers-and-stagnation]]` — kill switches, watchdogs, and runaway-loop prevention
