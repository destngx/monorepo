# GW-VALIDATE-102: Response Schema Validation

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Response validation and schema enforcement
- **Feature** : Enforce response schema correctness and type safety before returning to clients
- **Rationale** : Ensure all responses match declared Pydantic models; catch schema drift early; maintain contract with API consumers

---

### Input

- **Data / Files** : `[[../../../src/models.py]]`, `[[../../../src/main.py]]`, `[[../plan/input-output-validation.md]]`
- **Dependencies** : GW-VALIDATE-101 (request validation must exist first)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Validate `ExecuteResponse` structure and types before return
  - Validate `InvalidateResponse` structure and types before return
  - Validate status endpoint response structure
  - Check that run_id and thread_id are valid UUID format
  - Check that status field is in allowed set ["pending", "running", "completed", "failed"]
  - Create `tests/test_response_validation.py` with TDD RED tests

- **Out of Scope**:
  - Response content validation (e.g., checking workflow_id against a registry)
  - Response caching or transformation
  - Error response standardization (GW-VALIDATE-103)

- **Max Increment**: Add response model validators and status endpoint schema check

---

### Approach

**TDD Red-Green-Refactor Flow**:

1. **RED**: Write failing tests for invalid response schemas
2. **GREEN**: Add Pydantic field validators to response models
3. **REFACTOR**: Extract UUID and enum validation into reusable helpers

**Implementation Steps**:

1. Create `tests/test_response_validation.py` with 6 failing RED tests:
   - Test 1: ExecuteResponse with invalid run_id (not UUID) → expect validation error on model construction
   - Test 2: ExecuteResponse with invalid thread_id (not UUID) → expect validation error on model construction
   - Test 3: ExecuteResponse with missing status field → expect validation error
   - Test 4: InvalidateResponse with invalid status (not in ["invalidated", "not_found"]) → expect validation error
   - Test 5: Status endpoint response with status not in valid set → expect validation error
   - Test 6: ExecuteResponse with null/None workflow_id → expect validation error

2. Modify `src/models.py`:
   - Add `from pydantic import field_validator` import if not present
   - Update `ExecuteResponse` class validators:
     - `validate_run_id(self)` → check UUID format
     - `validate_thread_id(self)` → check UUID format
     - `validate_status(self)` → check status in ["pending", "running", "completed", "failed"]
   - Update `InvalidateResponse` class validators:
     - `validate_status(self)` → check status in ["invalidated", "not_found"]

3. Create `src/response_schema.py` (optional, can extend validation.py):
   - `validate_uuid_format(value, field_name)` → check valid UUID format
   - `validate_status_enum(value, allowed_statuses)` → check status in allowed set

4. Modify `src/main.py`:
   - Verify all handlers instantiate response models correctly
   - Add defensive checks before returning responses (optional but recommended)

5. Run full test suite:
   - `pytest tests/test_response_validation.py -v` → all 6 tests pass
   - `pytest tests/ -v` → all 81+ tests pass, zero regressions

**Files to Modify/Create**:

- `src/models.py` — add validators to response model classes
- `src/response_schema.py` (optional) — extract response validation helpers
- `tests/test_response_validation.py` — create with 6 RED tests
- `src/main.py` — verify handler implementations return correct response models

---

### Expected Output

- **Deliverable** : Response schema validation layer with Pydantic validators
- **Format** : Python modules (models.py updates, optional response_schema.py) + test suite
- **Evidence** : 6 passing tests, zero regressions, all responses correctly typed

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-VALIDATE-102-FUNC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-102-DOC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-102-SCHEMA.md]]

---

### References

[[../plan/input-output-validation.md]] - Full plan for validation strategy.

[[../../../tests/test_execution.py]] - Reference for existing test patterns.

[[../MOCK/GW-VALIDATE-101.md]] - Request validation task (prerequisite).
