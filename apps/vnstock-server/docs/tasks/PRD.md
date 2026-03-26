# Product Requirements Document (PRD) - Vnstock Server Refactoring

## Overview

The `vnstock-server` is currently a single-file FastAPI application (`main.py`) containing many endpoints. This makes maintenance difficult. We need to refactor it using **Vertical Slice Architecture**, separating concerns into cohesive modules.

## Goals

- Separate endpoints into vertical slices (e.g., Stocks/Listing, Quote/Trading, Finance, Company, Health).
- Improve code readability and maintainability.
- Maintain existing functionality, caching, and rate limiting.

## Proposed Structure

The project will be organized into a `modules` directory, with each module containing its own router and logic.

```text
src/
├── main.py (Entry point)
├── config.py
├── cache.py
├── rate_limiter.py
└── modules/
    ├── __init__.py
    ├── shared/ (Common utilities between modules)
    ├── listing/
    ├── stocks/ (Quote & Trading)
    ├── finance/
    ├── company/
    └── health/
```

## Task List (Progress Tracking)

- [x] Initialize `modules` directory structure.
- [x] Create shared dependencies/utilities in `modules/shared/deps.py`.
- [x] Implement `listing` slice.
- [x] Implement `stocks` slice (Quote/Trading).
- [x] Implement `finance` slice.
- [x] Implement `company` slice.
- [x] Implement `health` slice.
- [x] Refactor `main.py` to use `APIRouter` from modules.
- [x] Verify functionality with existing tests.
