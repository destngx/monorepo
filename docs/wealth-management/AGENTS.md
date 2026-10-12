# Wealth Management: Engineering & Agent Standards

This document defines the core standards for the Wealth Management platform, covering the Go backend and SvelteKit frontend.

**Scope and status of features**: [`REQUIREMENTS.md`](./REQUIREMENTS.md) is the source of truth.

## 📦 Layout

One Nx project, `wealth-management`, at `apps/wealth-management/`:

- Go engine (Fiber) at the project root, API under `/api`, MCP server over stdio (stdout is reserved for JSON-RPC; logs go to stderr).
- SvelteKit UI in `ui/`, built with `adapter-static` (base `/ui`) and served by the engine at `/ui`. One process, one origin in production.
- Dev: `pnpx nx run wealth-management:serve` (air + vite with `/api` proxied to `:8080`). Prod-like: `pnpx nx run wealth-management:start` (builds the UI, then runs the engine).
- Google OAuth: `pnpm run auth:sheets`; check with `pnpx nx run wealth-management:verify-auth`.

## 🏗️ Backend: Go (Hexagonal Architecture)

We use a strict **Hexagonal (Ports & Adapters)** architecture.

### 📁 Layer Responsibilities

- **`domain/`**: Pure business logic and entities. Zero-dependency.
- **`port/`**: Interfaces defining the "contracts" (Primary for input, Secondary for output).
- **`service/`**: Business use-case orchestrators.
- **`adapter/`**: Infrastructure implementations (DB, Cache, External APIs).

### 🏷️ Naming Conventions

- **Domain**: Prefer business naming (`accounts`, not `accounts_sheet`).
- **Adapters**: Name by _capability_, not _technology_ (e.g., `adapter/cache` instead of `adapter/redis`).

## 🎨 Frontend: SvelteKit (Feature-Sliced Design)

The frontend follows **Feature-Sliced Design (FSD)** to maintain scalability.

### 📁 FSD Structure

- **`src/features/`**: Complex logic (e.g., `accounts-ledger`, `transaction-entry`).
- **`src/entities/`**: Domain models and data logic.
- **`src/shared/`**: Reusable UI components, utils, and themes.
- **`src/routes/`**: Pages only. The UI is a client-rendered SPA (`ssr = false`); all data comes from the Go `/api`.

### 💅 UI & Styling

- **Glassmorphism**: The primary aesthetic theme.
- **Tailwind CSS**: Used for all styling. Avoid vanilla CSS where possible.

## 🧪 Testing & Workflow

- **Standard**: TDD + BDD.
- **Tooling**: pnpm only (`pnpm run`, `pnpx nx`).
- **Backend Test**: `pnpx nx run wealth-management:test`
- **Frontend Test**: `pnpx nx run wealth-management:test-ui` (type check: `check-ui`, lint: `lint-ui`)
- **Live tests**: only against a disposable copy of the Google Sheet, never the production sheet (REQUIREMENTS.md §14).
- **Security**: Never read or print from `.env.local` in code; it is for runtime only.
