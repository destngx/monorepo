# Plan

Purpose: capture architecture sequencing, trade-offs, and boundary decisions.

## Use for

- system context and platform boundary decisions
- architecture trade-offs that affect delivery order
- decisions that must be recorded before implementation

## Rules

- keep entries short
- link back to the relevant architecture spec
- record why a boundary exists and what it protects
- prefer decisions over narrative
- update this layer before any implementation work that changes a boundary

## See also

- `[[../README]]`
- `[[../tasks/README]]`
- `[[../verification/README]]`

## Plan files

- `[[platform-boundary-and-fixed-stack]]` — fixed stack, platform layers, and external contract
- `[[runtime-execution-flow]]` — request path, validation, execution, and SSE boundary
- `[[tenant-isolation-and-scoping]]` — tenant, workflow, and thread isolation rules
