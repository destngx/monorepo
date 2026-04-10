# GW-MVP-RUNTIME-207: Workflow Update Semantics

**Objective**: Make workflow updates safe by enforcing immutable field rules, version tracking, and change propagation semantics.

**Phase**: [MVP]

## Requirements

### Functional

- Validate immutable fields are not modified
- Require permission checks for updates
- Create version history entries on update
- Trigger workflow recompilation when definitions change
- Notify dependent workflows when changes occur
- Write audit log entries for update actions

### Non-Functional

- Update behavior must be explicit and reversible in the docs
- Version history must remain a first-class contract

## Acceptance Criteria

- [ ] Immutable workflow fields are protected
- [ ] Update requests require proper authorization
- [ ] Version history is created for updates
- [ ] Definition changes trigger recompilation notes
- [ ] Dependent workflow notification is documented
- [ ] Audit logging is required for updates

## Related Requirements

- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant, workflow, and thread isolation must apply to workflow execution, Redis state, skill caches, and kill switches

## Deliverables

1. `src/main.py` update semantics notes
2. `tests/test_workflow_update.py` or equivalent coverage

## Verification Criteria

[[../../verification/MVP/VERIFY-GW-RUNTIME-207.md]]
