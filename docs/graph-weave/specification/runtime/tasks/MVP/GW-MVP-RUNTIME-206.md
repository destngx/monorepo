# GW-MVP-RUNTIME-206: Workflow Read and List Behavior

**Objective**: Define the MVP behavior for workflow list/read endpoints, including filtering, permissions, and cache expectations.

**Phase**: [MVP]

## Requirements

### Functional

- Add permission checks for workflow reads and listings
- Return only workflows the caller may see
- Support pagination for list queries
- Support sorting by name, created_at, and updated_at
- Support full-text search on workflow name/description
- Include version history when requested
- Include execution statistics when requested
- Add cache TTL expectations for list/read responses

### Non-Functional

- Read/list behavior must be deterministic and tenant-safe
- Cache behavior must remain explicit and documented

## Acceptance Criteria

- [ ] Workflow list enforces permission checks
- [ ] Workflow list supports pagination
- [ ] Workflow list supports sorting
- [ ] Workflow list supports full-text search
- [ ] Workflow read enforces permission checks
- [ ] Workflow read can return version history when requested
- [ ] Workflow read can include execution stats when requested
- [ ] Cache TTL requirements are documented

## Related Requirements

- FR-RUNTIME-001 [MOCK,MVP]: Workflow submission must be validated before execution
- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant, workflow, and thread isolation must apply to workflow execution, Redis state, skill caches, and kill switches

## Deliverables

1. `src/main.py` read/list behavior notes
2. `tests/test_workflow_read_list.py` or equivalent coverage

## Verification Criteria

[[../../verification/MVP/VERIFY-GW-RUNTIME-206.md]]
