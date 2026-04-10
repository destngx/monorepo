# GW-MVP-RUNTIME-208: Workflow Delete Semantics

**Objective**: Make workflow deletion safe by enforcing authorization, dependency checks, and soft-delete semantics.

**Phase**: [MVP]

## Requirements

### Functional

- Require permission checks for deletion
- Check whether the workflow is referenced by other workflows
- Check whether active executions are still using the workflow
- Add audit log entries for deletion
- Prefer soft delete behavior with history retention
- Notify users who saved the workflow as a template

### Non-Functional

- Delete behavior must be conservative and traceable
- Hard-delete semantics should remain out of scope for MVP unless explicitly documented

## Acceptance Criteria

- [ ] Delete requests require authorization
- [ ] Referenced workflows cannot be deleted silently
- [ ] Active executions block deletion or require explicit handling
- [ ] Audit logging is required for deletion
- [ ] Soft-delete behavior is documented
- [ ] Template notification requirement is documented

## Related Requirements

- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant, workflow, and thread isolation must apply to workflow execution, Redis state, skill caches, and kill switches

## Deliverables

1. `src/main.py` delete semantics notes
2. `tests/test_workflow_delete.py` or equivalent coverage

## Verification Criteria

[[../../verification/MVP/VERIFY-GW-RUNTIME-208.md]]
