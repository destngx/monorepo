## ⚡ CURRENT PHASE — READ THIS FIRST

Find the line with the below Phase and only read the line after that

> **Phase 3: Vnstock Python Server is 0% complete.**
>
> **Active Phase: PHASE 3 — Vnstock Python Server**
> **Next task: Scaffold apps/vnstock-server and initialize Python environment**
> **Instructions: See `prompt.md` §1 for core loop steps.**

## Status Summary

| Phase                    | Tasks | Done | Status      |
| ------------------------ | ----- | ---- | ----------- |
| Phase 1: Initialization  | 4     | 4    | COMPLETE    |
| Phase 2: Remove Task App | 4     | 4    | COMPLETE    |
| Phase 3: Vnstock Server  | 4     | 0    | IN PROGRESS |

## Implementation Plan

Remove the task management application, its associated libraries, and E2E tests from the monorepo to declutter the codebase.

### Proposed Changes

#### Applications and Libraries

- [DELETE] `apps/task-management`
- [DELETE] `apps/task-management-e2e`
- [DELETE] `libs/task-management`

#### Configuration Updates

- [MODIFY] `tsconfig.base.json`: Remove path aliases for `@task-management/config` and `@task-management/queue`.
- [MODIFY] `package.json`: Remove unused dependencies:
  - `@nestjs/bull`
  - `@nestjs/common`
  - `@nestjs/config`
  - `@nestjs/core`
  - `@nestjs/platform-express`
  - `@nestjs/platform-fastify`
  - `@nestjs/schematics`
  - `@nestjs/testing`
  - `bull`
  - `reflect-metadata`
  - `rxjs`
- [MODIFY] `docs/plan/ARCHITECTURE.md`: Remove references to the Task Management app.

### Verification Plan

#### Automated Tests

- Run `bun nx lint wealth-management` to ensure no broken imports in the primary app.
- Run `bun nx lint cloudinary-photos-app` to ensure no broken imports.
- Run `bun install` to update the lockfile and verify dependency integrity.
- Run `bun nx show projects` to verify the projects are removed from the Nx graph.

#### Manual Verification

- Verify that the `apps` and `libs` directories no longer contain `task-management` folders.

## Phase 2 Checklist

- [x] Task 2.1: Delete `apps/task-management` and `apps/task-management-e2e`
- [x] Task 2.2: Delete `libs/task-management`
- [x] Task 2.3: Update config files (`tsconfig.base.json`, `package.json`)
- [x] Task 2.4: Verify removal and update lockfile

## Phase 3: Vnstock Python Server Implementation Plan

### Proposed Changes

#### Applications and Libraries

- [NEW] `apps/vnstock-server/` - Directory for the new python application.
- [NEW] `apps/vnstock-server/src/main.py` - FastAPI entrypoint wrapping the `vnstock` library.
- [NEW] `apps/vnstock-server/requirements.txt` - Python dependencies (`fastapi`, `uvicorn`, `vnstock`).

#### Configuration Updates

- [NEW] `apps/vnstock-server/project.json` - Nx configuration to define executors for the Python app (`serve`, `install`).
- [MODIFY] `docs/plan/ARCHITECTURE.md` - Document the inclusion of the Python microservice.

### Verification Plan

#### Automated Tests

- Run `cd apps/vnstock-server && pip install -r requirements.txt` (or via Nx target) to verify dependencies install cleanly.
- Run `bun nx serve vnstock-server` and perform a health check to verify it boots.
- Add simple unit tests (e.g., using `pytest`) and a `test` target to `project.json`, verifiable with `bun nx test vnstock-server`.

#### Manual Verification

- Run `bun nx show projects` to verify Nx properly recognizes the new `vnstock-server` project.
- Perform a manual `curl` request to the new FastAPI endpoints (e.g., `curl http://localhost:8000/api/health` and a data-fetching endpoint) from the terminal to ensure `vnstock` connects successfully.

## Phase 3 Checklist

- [x] Task 3.1: Scaffold `apps/vnstock-server` and initialize Python environment (`requirements.txt`, `src/main.py`, `project.json`).
- [x] Task 3.2: Implement FastAPI endpoints to wrap core `vnstock` functionality (e.g. historical data, stock listing).
- [x] Task 3.3: Configure Nx integration and verify workspace (`bun nx show projects`).
- [x] Task 3.4: Update `ARCHITECTURE.md` and manually test endpoints.
- [ ] Task 3.5: Implement Stock Analysis service in `libs/wealth-management` consuming `vnstock-server`.
- [ ] Task 3.6: Upgrade vnstock to vnstock 3.5.0
- [ ] Task 3.7: Final verification and cleanup.
