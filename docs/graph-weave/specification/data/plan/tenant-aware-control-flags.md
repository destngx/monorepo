# Tenant-Aware Control Flags

Purpose: record the blast-radius rules for kill switches and active-thread tracking.

## Scope

- tenant kill switches
- workflow kill switches
- thread kill switches
- active-thread auditing

## References and context

- `[[../redis-namespace-design]]` defines kill-switch keys and active-thread tracking. This reference matters because control flags are part of the data contract, not just runtime behavior.
- `[[../../architecture/plan/tenant-isolation-and-scoping]]` defines the tenant/workflow/thread isolation model. This reference is included because the data layout must preserve that isolation at storage level.

## Decisions

- Encode blast radius in key naming.
- Keep active-thread data tenant-scoped.
- Ensure one tenant's stop flag cannot affect another tenant.
- Make control flags easy to audit in Redis.
- Use kill-switch for tenant-wide stop, rate-limit for workflow throttling, feature-flag for tenant-wide behavior, and quota for tenant budget control.
- Treat TTL as part of the control contract so stale flags expire predictably.

## Definition of done

- Control flags are mapped to their scopes.
- Active-thread tracking remains tenant-aware.
- The storage plan explains how interruption stays isolated.
- The blast radius of each flag type is documented clearly.

## Next links

- `[[redis-namespace-layout-and-scope]]`
- `[[compiled-graph-and-checkpoint-storage]]`
