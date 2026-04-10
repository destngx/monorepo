# GW-MVP-RUNTIME-204: Execution Endpoint Hardening

**Objective**: Harden `POST /execute` and `GET /execute/{run_id}/status` so execution is authenticated, pre-created, validated, and returns the correct MVP lifecycle semantics.

**Phase**: [MVP]

## Requirements

### Functional

- Authenticate the caller before workflow execution starts
- Verify the caller may execute the target workflow
- Reject archived or deprecated workflows before execution
- Validate input against the workflow input schema
- Require pre-created workflows; do not auto-create at execution time
- Return actual execution status from the executor when available
- Return 404 for non-existent runs
- Support queued/validating/pending/running status flow in the execution path
- Add execution timeout handling with a configurable default

### Non-Functional

- Execution guard checks must be explicit and easy to audit
- Status behavior must match the MVP request-lifecycle contract

## Acceptance Criteria

- [ ] Execution request requires authn/authz before graph launch
- [ ] Archived/deprecated workflows cannot execute
- [ ] Invalid input is rejected before execution begins
- [ ] Pre-created workflow requirement is enforced
- [ ] Status endpoint returns 404 for missing runs
- [ ] Executor status is surfaced instead of fixed mock status
- [ ] Timeout handling is documented and testable

## Related Requirements

- FR-RUNTIME-001 [MOCK,MVP]: Workflow submission must be validated before execution
- FR-RUNTIME-002 [MOCK,MVP]: Workflow submission must return a run id for later status access
- FR-RUNTIME-003 [MOCK,MVP,FULL]: A separate SSE status request must stream structured events for the submitted run
- FR-RUNTIME-004 [MVP,FULL]: Checkpoints and active thread state must survive interruptions and completion

## Deliverables

1. `src/main.py` execution guards and status behavior
2. `tests/test_execute_endpoint.py` or equivalent coverage for execution lifecycle guards
3. `tests/test_status_endpoint.py` or equivalent coverage for missing run handling

## Verification Criteria

[[../../verification/MVP/VERIFY-GW-RUNTIME-204.md]]
