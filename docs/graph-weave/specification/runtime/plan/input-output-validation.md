# GW-VALIDATE Plan: Input/Output Validation

**Date**: 2026-04-09  
**Component**: Runtime  
**Phase**: MOCK  
**Objective**: Add comprehensive request input validation, response schema validation, and error response validation to the mock application before MVP phase hardening.

---

## Context

The MOCK phase has completed 16 core tasks but is missing an essential cross-cutting concern: **input/output validation**. Currently:

- `ExecuteRequest` has Pydantic models but minimal validation rules
- `ExecuteResponse` returns data without schema enforcement checks
- Error responses are unstructured
- No distinction between client errors (4xx) and server errors (5xx)
- Invalid requests pass silently (no early rejection)

This gap will compound in MVP as real external systems are integrated. Validation must be hardened in MOCK before moving to real backends.

---

## Design Decisions

### Request Validation Strategy

**Scope**: Validate all incoming request payloads before business logic execution.

- **What to validate**: `tenant_id`, `workflow_id`, `run_id`, `input` dict structure, `skill_id`, `reason`
- **Rules**:
  - `tenant_id`: non-empty string, no whitespace, max 128 chars
  - `workflow_id`: non-empty string, no whitespace, max 128 chars
  - `run_id` (optional): UUID format if provided
  - `input` (ExecuteRequest): dict, can be empty, keys must be strings
  - `skill_id`: non-empty string, no whitespace, max 128 chars
  - `reason` (InvalidateRequest): non-empty string, max 256 chars
- **Failure mode**: Return HTTP 422 Unprocessable Entity with field-level error details
- **Validation location**: Pydantic validators in `models.py` + endpoint-level guards

### Response Validation Strategy

**Scope**: Ensure all responses match their declared schemas.

- **What to validate**:
  - `ExecuteResponse`: all fields present, types correct, run_id/thread_id are valid UUIDs
  - `InvalidateResponse`: all fields present, status in ["invalidated", "not_found"]
  - Status endpoint response: run_id matches path param, status in valid states
- **Rules**: Response models are Pydantic, so validation happens on model construction
- **Failure mode**: Log validation errors, return 500 with error details
- **Validation location**: Response model instantiation in handlers + type hints

### Error Response Validation Strategy

**Scope**: Standardize error responses across all endpoints.

- **Error format**:
  ```json
  {
    "error": "ErrorType",
    "message": "Human-readable description",
    "details": { "field": "reason" },
    "status_code": 422
  }
  ```
- **Error types**:
  - `ValidationError` (422): Request data invalid
  - `NotFoundError` (404): Resource not found
  - `InternalServerError` (500): Unexpected server error
  - `ConfigurationError` (500): Config validation failed
- **Validation location**: Exception handlers + custom error response builder

---

## Task Breakdown

### GW-VALIDATE-101: Request Input Validation

**Deliverable**: Pydantic validators for all request models + endpoint-level guards.

- Add field-level validators to `ExecuteRequest`, `InvalidateRequest` in `models.py`
- Add `validation.py` module with helper functions
- Return 422 on validation failure with field details
- 3 tests: valid request, invalid tenant_id, invalid workflow_id

### GW-VALIDATE-102: Response Schema Validation

**Deliverable**: Response schema enforcement + type hints.

- Verify all endpoint handlers return correct response models
- Add response schema documentation
- 3 tests: valid response structure, UUID format checks, missing required fields

### GW-VALIDATE-103: Error Response Standardization

**Deliverable**: Unified error response format + exception handlers.

- Create `errors.py` module with custom exception classes
- Add FastAPI exception handlers for each error type
- Standardize error response body + HTTP status codes
- 3 tests: validation error response, not found response, server error response

---

## Files to Create/Modify

| File                                | Action | Purpose                               |
| ----------------------------------- | ------ | ------------------------------------- |
| `src/models.py`                     | Modify | Add Pydantic field validators         |
| `src/validation.py`                 | Create | Validation helper functions           |
| `src/errors.py`                     | Create | Custom exception classes              |
| `src/main.py`                       | Modify | Add exception handlers                |
| `tests/test_request_validation.py`  | Create | TDD RED tests for input validation    |
| `tests/test_response_validation.py` | Create | TDD RED tests for response validation |
| `tests/test_error_response.py`      | Create | TDD RED tests for error responses     |

---

## Success Criteria

- [ ] All request fields validated with clear error messages
- [ ] All responses match schema before returning to client
- [ ] All errors return standardized format with correct HTTP status
- [ ] 9 new tests added (3 per task), all passing
- [ ] Zero regressions in existing 69 tests
- [ ] Verification files created for each task (FUNC, DOC, SCHEMA)

---

## References

- `[[../README]]` - Runtime Living Spec
- `[[../../workflow-schema/WORKFLOW_JSON_SPEC.md]]` - Workflow schema contracts
- `[[../tasks/MOCK]]` - Existing MOCK phase tasks
