# Compiled Graph and Checkpoint Storage

Purpose: record how runtime artifacts are stored so reusable graph caches do not collide with live checkpoint state.

## Scope

- compiled graph cache keys
- checkpoint storage keys
- TTL and cacheability intent
- resume-friendly storage boundaries

## References and context

- `[[../redis-namespace-design]]` distinguishes compiled graphs from checkpoints and shows that both must be independently addressable. This reference is included because the plan depends on that separation to keep recovery predictable.
- `[[../verification/README]]` covers migration and compatibility checks. This context matters because storage changes need explicit verification when the namespace shape changes.

## Decisions

- Store compiled graphs separately from checkpoints.
- Treat compiled graphs as cacheable artifacts.
- Keep checkpoint storage distinct so resume logic stays clean.
- Preserve compatibility when namespace changes are unavoidable.

## Definition of done

- Graph cache and checkpoint storage are documented as separate concerns.
- TTL intent is visible where it affects operator behavior.
- The plan makes it hard to confuse live state with cached runtime artifacts.

## Next links

- `[[redis-namespace-layout-and-scope]]`
- `[[tenant-aware-control-flags]]`
