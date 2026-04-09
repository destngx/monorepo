# VERIFY-GW-VALIDATE-103-SCHEMA: Error Response Standardization Schema

## Task Reference

`[[../../tasks/MOCK/GW-VALIDATE-103.md]]`

## Schema Validation Checklist

### Error Response Schema (All Error Types)

- [x] **Base Error Schema**
  - `error`: string (error type name) ✓
  - `message`: string (human-readable description) ✓
  - `details`: array (error-specific details) ✓
  - `status_code`: integer (HTTP status code) ✓

- [x] **ValidationError Response**
  - status_code: 422 ✓
  - details structure: array of {field, message, type} ✓
  - Example: `{ "error": "ValidationError", "message": "Request validation failed", "details": [...], "status_code": 422 }` ✓

### Exception Handler Schema

- [x] **FastAPI Handler Registration**
  - @app.exception_handler(RequestValidationError) registered ✓
  - Handler catches Pydantic validation errors ✓
  - Handler returns JSONResponse with correct status code ✓

- [x] **Handler Response Format**
  - status_code matches exc.status_code (422) ✓
  - content includes error, message, details, status_code ✓
  - Headers include "content-type": "application/json" ✓

### OpenAPI Integration

- [x] **422 Response Defined**
  - In POST /execute endpoint ✓
  - In POST /invalidate endpoint ✓
  - Shows ValidationError schema ✓

### Type Checking

- [x] All response fields have proper types ✓
- [x] Exception handler has correct async/await types ✓
- [x] No type errors (verified through tests) ✓

---

## Status

- **Status**: Pass
- **Schema Validation**: Complete
- **Type Checking**: Verified

---

## Sign-Off

**Verified By**: Sisyphus (Agent)  
**Date**: 2026-04-09  
**Notes**: All schemas validated and verified.
