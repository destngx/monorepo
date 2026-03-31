# Specification: Code Modularization and Testability Strategy

**Author**: Engineering  
**Intended Audience**: Wealth Management Engine/Dashboard Developers  
**Status**: Active Standard  
**Goal**: Keep code cognitively manageable and easy to test by enforcing small vertical slices, clear boundaries, and strict test ownership.

---

## 1. Design Principle

Build by **use-case slices** with small files/functions, then compose at the boundary (`main.go` / route registration).  
Do not let handlers/services become "god files".

---

## 2. Size and Complexity Budgets

- **File size target**: <= 200 lines (hard cap: 300).
- **Function size target**: <= 40 lines (hard cap: 60).
- **Single responsibility**: each file should serve one clear concern.
- If a hard cap is exceeded, refactor is required in the same task unless explicitly approved.

---

## 3. Folder and Slice Strategy (Go)

Keep hexagonal architecture, but split by capability/use-case inside each layer.

- `adapter/fiber/<domain>/...`:
  - one endpoint concern per file (`stream_handler.go`, `json_handler.go`, etc.).
- `service/<domain>/...`:
  - one business capability per file (`structured_json_service.go`, `ticker_service.go`, etc.).
- `port/...`:
  - small, stable interfaces only.
- `domain/...`:
  - business types and invariants only; no transport-specific fields.

Use shared helpers only when reused by >=2 slices; otherwise keep logic local to maintain clarity.

---

## 4. DTO and Mapping Boundaries

- Separate transport DTOs from domain models.
- Keep request/response parsing and validation in adapter DTO/mapper files.
- Keep services pure: accept domain-oriented inputs and return domain models.

This prevents handler bloat and makes unit tests deterministic.

---

## 5. Test Strategy Alignment

- Test location must mirror production ownership.
- Unit tests stay next to the file they validate.
- Integration/e2e HTTP tests stay in API adapter package (`adapter/fiber`) with `TestE2E*` naming.
- E2E tests must assert response meaning (schema/content), not status-only.
- Live e2e policy is strict: failures must fail the run (non-zero exit).

---

## 6. Refactor Workflow (Required)

For oversized files:

1. Add/lock behavior with tests first (TDD/BDD).
2. Extract one coherent slice at a time.
3. Keep old behavior unchanged while moving code.
4. Run `bunx nx run wealth-management-engine:test`.
5. Run `bunx nx run wealth-management-engine:test-e2e` for API-impacting changes.

---

## 7. Composition Root Rule

`main.go` should only:

- load config,
- wire dependencies,
- register routes,
- start runtime.

Move setup logic for each domain into dedicated bootstrap/wiring functions when growth continues.

---

## 8. Enforcement in Code Review

Every PR touching backend runtime should be checked for:

- file/function size caps,
- slice boundaries preserved,
- tests co-located and meaningful,
- no new god files,
- unchanged public API contract unless documented in sprint tasks.
