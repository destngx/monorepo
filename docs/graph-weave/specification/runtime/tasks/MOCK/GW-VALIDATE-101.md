# GW-VALIDATE-101: Request Input Validation

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Request validation and error handling
- **Feature** : Reject invalid requests early with clear error messages before business logic executes
- **Rationale** : Prevent invalid data from reaching downstream systems and mock logic; ensure consistent error reporting across all endpoints
- **Design Note** : ExecuteRequest contains only (tenant_id, workflow_id, input). run_id is server-generated and appears only in responses. See delta-changes for rerun debate outcome.

---

### Input

- **Data / Files** : `[[../../../src/models.py]]`, `[[../../../src/main.py]]`, `[[../plan/input-output-validation.md]]`
- **Dependencies** : None (foundation task)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Validate all fields in `ExecuteRequest` (tenant_id, workflow_id, input)
  - Validate all fields in `InvalidateRequest` (tenant_id, skill_id, reason)
  - Return HTTP 422 Unprocessable Entity with field-level error details
  - Create `src/validation.py` with reusable validators
  - Create `tests/test_request_validation.py` with TDD RED tests

- **Out of Scope**:
  - Response validation (GW-VALIDATE-102)
  - Error response standardization (GW-VALIDATE-103)
  - Rate limiting or quota enforcement
  - Async validation or external validation services
  - run_id field validation in requests (removed per API design)

- **Max Increment**: Add field validators to models and validation helpers module

---

### Approach

**TDD Red-Green-Refactor Flow**:

1. **RED**: Write failing tests for invalid requests (empty tenant_id, invalid workflow_id, empty skill_id)
2. **GREEN**: Add Pydantic field validators to models.py using `@field_validator` decorators
3. **REFACTOR**: Extract common validation logic into `src/validation.py` helper functions

**Implementation Steps**:

1. Create `tests/test_request_validation.py` with 7 failing RED tests:
   - Test 1: ExecuteRequest with empty tenant_id → expect 422 validation error
   - Test 2: ExecuteRequest with empty workflow_id → expect 422 validation error
   - Test 3: InvalidateRequest with empty skill_id → expect 422 validation error
   - Test 4: InvalidateRequest with empty reason → expect 422 validation error
   - Test 5: InvalidateRequest with reason > 256 chars → expect 422 validation error
   - Test 6: POST /execute endpoint returns 422 on invalid tenant_id
   - Test 7: POST /invalidate endpoint returns 422 on invalid skill_id

2. Create `src/validation.py` with helper functions:
   - `validate_resource_id(value, field_name, max_len=128)` → check non-empty, no whitespace, max length
   - `validate_optional_uuid(value, field_name)` → check UUID format if provided (for response validation)

3. Modify `src/models.py`:
   - Remove run_id field from ExecuteRequest (API design change)
   - Add `from pydantic import field_validator` import
   - Add validators to `ExecuteRequest` class:
     - `validate_tenant_id(self)` → calls `validate_resource_id(self.tenant_id, "tenant_id")`
     - `validate_workflow_id(self)` → calls `validate_resource_id(self.workflow_id, "workflow_id")`
   - Add validators to `InvalidateRequest` class:
     - `validate_tenant_id(self)` → calls `validate_resource_id(self.tenant_id, "tenant_id")`
     - `validate_skill_id(self)` → calls `validate_resource_id(self.skill_id, "skill_id")`
     - `validate_reason(self)` → check non-empty, max 256 chars

4. Modify `src/main.py`:
   - Add exception handler for `ValidationError` from Pydantic
   - Return HTTP 422 with error details in response body

5. Run full test suite:
   - `pytest tests/test_request_validation.py -v` → all 7 tests pass
   - `pytest tests/ -v` → all 87+ tests pass, zero regressions

**Files to Modify/Create**:

- `src/models.py` — add field validators, remove run_id from ExecuteRequest
- `src/validation.py` — create with helper functions
- `src/main.py` — add exception handler for ValidationError
- `tests/test_request_validation.py` — create with 7 RED tests

---

### Expected Output

- **Deliverable** : Request validation layer with Pydantic validators + validation helpers
- **Format** : Python modules (models.py, validation.py, main.py) + test suite
- **Evidence** : 7 passing tests, zero regressions (87 total tests passing), lsp_diagnostics clean
- **Real Use Case** : Tests use financial research pipeline values (hedge_fund_research_desk, quant-research:v3.0.0, web_search)

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-VALIDATE-101-FUNC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-101-DOC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-101-SCHEMA.md]]

---

### Design Decisions & Debate

**Decision**: Removed run_id from ExecuteRequest. run_id is server-generated only.

**Rationale**:

- API contract clarity: clients don't provide execution IDs, server does
- Prevents confusion about client-provided vs server-generated IDs
- Simplifies validation logic and request model

**Debate History** (See `[[../../../../../delta-changes.md]]`):

- Initial design: optional run_id in request for recovery/rerun flows
- Feedback: "run_id should NOT be in request, only response"
- Resolution: Removed from ExecuteRequest entirely
- Trade-off: Rerun/recovery now requires alternative design (future task)
- Impact: Removed 4 tests (test*execute_request_invalid_run_id_uuid_format, test_rerun*\* tests)

---

### References

[[../plan/input-output-validation.md]] - Full plan for validation strategy.

[[../../../tests/test_request_validation.py]] - Test implementation with real use case values.

[[../../../src/main.py]] - Current endpoint implementations.
