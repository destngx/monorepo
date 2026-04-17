# Wealth Management: Engineering & Agent Standards

This document defines the core standards for the Wealth Management platform, covering the Go backend and SvelteKit frontend.

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
- **`src/routes/`**: Pages and API route handlers.

### 💅 UI & Styling

- **Glassmorphism**: The primary aesthetic theme.
- **Tailwind CSS**: Used for all styling. Avoid vanilla CSS where possible.

## 🧪 Testing & Workflow

- **Standard**: TDD + BDD.
- **Backend Test**: `bunx nx run wealth-management-engine:test`
- **Frontend Test**: `bunx nx run wealth-management-dashboard:test`
- **Security**: Never read or print from `.env.local` in code; it is for runtime only.
