# Plan

Purpose: capture data-namespace sequencing, storage trade-offs, and migration decisions.

## Use for

- Redis namespace evolution
- key design trade-offs
- data migration and compatibility decisions

## Rules

- keep entries short and concrete
- document compatibility implications when keys or scopes change
- update before any storage-shape change is implemented

## See also

- `[[../README]]`
- `[[../tasks/README]]`
- `[[../verification/README]]`

## Plan files

- `[[redis-namespace-layout-and-scope]]` — workflow, checkpoint, skill, and kill-switch key structure
- `[[compiled-graph-and-checkpoint-storage]]` — cache/checkpoint separation and TTL intent
- `[[tenant-aware-control-flags]]` — tenant/workflow/thread blast-radius rules
