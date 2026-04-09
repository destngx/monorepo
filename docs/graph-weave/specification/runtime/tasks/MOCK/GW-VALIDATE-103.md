# GW-VALIDATE-103: Error Response Standardization

### Metadata

- **Phase ID** : MOCK
- **Risk Level** : Medium
- **Status** : Completed
- **Estimated Effort**: M
- **Assigned Agent** : Sisyphus

---

### Context

- **Bounded Context** : Error handling and response consistency
- **Feature** : Standardize error responses across all endpoints with consistent schema, HTTP status codes, and error types
- **Rationale** : Clients need predictable error format; MVP and FULL phases depend on uniform error contracts; enables structured logging and error recovery

---

### Input

- **Data / Files** : `[[../../../src/main.py]]`, `[[../plan/input-output-validation.md]]`, `[[../../../src/models.py]]`
- **Dependencies** : GW-VALIDATE-101, GW-VALIDATE-102 (request/response validation must exist)
- **External Systems**: None

---

### Scope

- **In Scope** :
  - Create custom exception classes: `ValidationError`, `NotFoundError`, `InternalServerError`, `ConfigurationError`
  - Standardize error response body: `{ error: str, message: str, details: dict, status_code: int }`
  - Add FastAPI exception handlers for each error type
  - Return correct HTTP status codes: 422 (validation), 404 (not found), 500 (server errors)
  - Create `src/errors.py` module with exception classes and helpers
  - Create `tests/test_error_response.py` with TDD RED tests

- **Out of Scope**:
  - Rate limit errors or quota errors (defer to FULL)
  - Error logging/observability (GW-ARCH-101 covers logging)
  - Error retry logic or circuit breakers

- **Max Increment**: Add error module with 4 exception classes + 3 FastAPI handlers

---

### Approach

**TDD Red-Green-Refactor Flow**:

1. **RED**: Write failing tests for error scenarios
2. **GREEN**: Create `src/errors.py` with exceptions and handlers
3. **REFACTOR**: Add handler registration to FastAPI app

**Implementation Steps**:

1. Create `tests/test_error_response.py` with 6 failing RED tests:
   - Test 1: ValidationError from invalid request → expect HTTP 422 with error field
   - Test 2: NotFoundError from missing resource → expect HTTP 404 with error field
   - Test 3: InternalServerError from unexpected condition → expect HTTP 500 with error field
   - Test 4: ConfigurationError from startup → expect HTTP 500 with error field
   - Test 5: Error response has required fields (error, message, details, status_code)
   - Test 6: Error response for missing workflow has "not_found" in error field

2. Create `src/errors.py` module:

   ```python
   class GraphWeaveError(Exception):
       """Base exception for GraphWeave"""
       pass

   class ValidationError(GraphWeaveError):
       """Raised when request validation fails"""
       def __init__(self, message: str, details: dict = None):
           self.message = message
           self.details = details or {}
           self.status_code = 422

   class NotFoundError(GraphWeaveError):
       """Raised when resource not found"""
       def __init__(self, message: str, details: dict = None):
           self.message = message
           self.details = details or {}
           self.status_code = 404

   class InternalServerError(GraphWeaveError):
       """Raised on unexpected server errors"""
       def __init__(self, message: str, details: dict = None):
           self.message = message
           self.details = details or {}
           self.status_code = 500

   class ConfigurationError(GraphWeaveError):
       """Raised on configuration validation failures"""
       def __init__(self, message: str, details: dict = None):
           self.message = message
           self.details = details or {}
           self.status_code = 500

   def error_response(error_type: str, message: str, details: dict, status_code: int) -> dict:
       """Build standardized error response"""
       return {
           "error": error_type,
           "message": message,
           "details": details,
           "status_code": status_code
       }
   ```

3. Modify `src/main.py`:
   - Import `from .errors import ...` exceptions and handlers
   - Add exception handlers:

     ```python
     @app.exception_handler(ValidationError)
     async def validation_error_handler(request, exc):
         return JSONResponse(
             status_code=exc.status_code,
             content=error_response("ValidationError", exc.message, exc.details, exc.status_code)
         )

     # Similar for NotFoundError, InternalServerError, ConfigurationError
     ```

   - Update existing handlers to raise exceptions (e.g., raise NotFoundError in status endpoint if run_id not found)

4. Run full test suite:
   - `pytest tests/test_error_response.py -v` → all 6 tests pass
   - `pytest tests/ -v` → all 87+ tests pass, zero regressions

**Files to Modify/Create**:

- `src/errors.py` — create with exception classes and helpers
- `src/main.py` — add exception handlers and raise exceptions in endpoints
- `tests/test_error_response.py` — create with 6 RED tests

---

### Expected Output

- **Deliverable** : Standardized error response layer with custom exceptions and FastAPI handlers
- **Format** : Python modules (errors.py, main.py updates) + test suite
- **Evidence** : 6 passing tests, zero regressions, consistent error format

---

### Verification Criteria

[[../../verification/MOCK/VERIFY-GW-VALIDATE-103-FUNC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-103-DOC.md]]

[[../../verification/MOCK/VERIFY-GW-VALIDATE-103-SCHEMA.md]]

---

### References

[[../plan/input-output-validation.md]] - Full plan for validation strategy.

[[../../../tests/test_errors.py]] - Reference for existing error test patterns.

[[../MOCK/GW-VALIDATE-101.md]] - Request validation task (prerequisite).

[[../MOCK/GW-VALIDATE-102.md]] - Response validation task (prerequisite).
