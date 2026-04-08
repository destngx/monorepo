# Tenant Isolation and Scoping

Purpose: record the tenant, workflow, and thread rules that keep execution isolated across state, caches, and kill switches.

## Scope

- tenant IDs
- workflow namespaces
- thread-scoped checkpoints
- skill-cache scoping
- kill-switch blast radius

## References and context

- `[[../multi-tenant]]` defines the authoritative tenant model and the isolation rules. This reference is included because it is the source of truth for all scoping decisions in the plan.
- `[[../architecture]]` links the architecture layer to the broader living spec. This reference matters because tenant isolation is a platform boundary, not a local implementation detail.

## Decisions

- Scope every request by tenant, workflow, and thread.
- Keep checkpoints thread-scoped.
- Make Redis namespaces tenant-aware for workflows, skills, and kill switches.
- Keep active-thread listings auditable per tenant.
- Route tenant identity from gateway to runtime without changing scope labels.
- Treat tenant, workflow, and thread as the only authoritative isolation levels.

## Definition of done

- Every isolation rule is tied to a concrete scope.
- The plan explains how state, caches, and kill switches stay separated.
- The runtime cannot infer a shared mutable scope across tenants.
- Each boundary has a documented failure mode and blast radius.

## Next links

- `[[platform-boundary-and-fixed-stack]]`
- `[[runtime-execution-flow]]`
