# Redis Namespace Layout and Scope

Purpose: record the Redis namespace structure so runtime state stays deterministic, inspectable, and tenant-aware.

## Scope

- workflow pointers and versions
- skill summary namespaces
- active-thread tracking
- kill-switch keys
- Redis lookup conventions

## References and context

- `[[../redis-namespace-design]]` defines the canonical keyspace, including workflow JSON, pointers, skill summaries, kill switches, active threads, and compiled graphs. This reference anchors the plan to the storage contract used by runtime engineers and operators.
- `[[../README]]` states that the data spec covers Redis namespaces, storage contracts, and state organization. This context is included so the plan remains data-layer specific.

## Decisions

- Keep workflow keys tenant-aware and versioned.
- Separate skill, checkpoint, active-thread, and kill-switch namespaces.
- Preserve predictable pointers for latest-version reads.
- Keep the keyspace readable for operator troubleshooting.
- Preserve the canonical {env}:{tenant_id}:{type}:{resource_id}:{field} shape where it helps explain scope.
- Keep the operator-facing namespace map stable across releases.

## Definition of done

- The key families are named and scoped.
- A reader can distinguish workflow, cache, checkpoint, and control data at a glance.
- The plan documents the blast-radius intent for each namespace.

## Next links

- `[[compiled-graph-and-checkpoint-storage]]`
- `[[tenant-aware-control-flags]]`
