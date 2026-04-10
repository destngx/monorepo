# GW-MVP-RUNTIME-205: Workflow Create Hardening

**Objective**: Harden workflow creation so new workflows are validated, unique, tenant-safe, and ready for later execution.

**Phase**: [MVP]

## Requirements

### Functional

- Add authentication and authorization for workflow creation
- Verify tenant ownership before creation proceeds
- Validate workflow definition schema, including DAG connectivity and node types
- Enforce name/ID uniqueness across the organization
- Persist workflows to PostgreSQL with audit logging
- Register workflows in the skill discovery system when needed

### Non-Functional

- Creation checks must fail fast with clear errors
- Persistence and audit requirements must be explicit

## Acceptance Criteria

- [ ] Workflow create requires authn/authz
- [ ] Tenant ownership is verified before creating a workflow
- [ ] Invalid workflow definitions are rejected
- [ ] Duplicate workflow identifiers are rejected
- [ ] Workflow persistence target is documented as PostgreSQL
- [ ] Audit logging requirement is documented
- [ ] Skill discovery registration requirement is documented

## Related Requirements

- FR-ARCH-012 [MOCK,MVP]: Request validation must happen before runtime execution
- FR-ARCH-013 [MOCK,MVP,FULL]: Tenant, workflow, and thread isolation must apply to workflow execution, Redis state, skill caches, and kill switches

## Deliverables

1. `src/main.py` workflow creation hardening notes
2. `tests/test_workflow_create.py` or equivalent coverage for create validation and uniqueness

## Verification Criteria

[[../../verification/MVP/VERIFY-GW-RUNTIME-205.md]]
