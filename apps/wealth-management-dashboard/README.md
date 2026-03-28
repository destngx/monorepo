# Wealth Management Dashboard (SvelteKit)

A high-performance Svelte-based frontend for the Wealth Management Platform, consuming the Go backend (`apps/wealth-management-engine`) API.

## Architecture

This is a **standalone SvelteKit application** with no separate `package.json`. All dependencies are managed via the root monorepo `package.json` and installed at the root level via `bun install`.

### Folder Structure (FSD)

```
src/
├── features/        # Feature-specific logic (e.g., accounts, transactions)
├── entities/        # Domain entities and models
├── shared/          # Shared UI components, utilities, constants
├── routes/          # SvelteKit routes and pages
├── lib/             # SvelteKit-provided library folder
├── app.d.ts         # Type definitions
└── app.html         # Root HTML template
```

## Development

### Start the dev server

```bash
# From root
bun run serve:wealth-management-dashboard

# Or directly from the app directory
cd apps/wealth-management-dashboard
bunx vite dev
```

The dashboard will be available at `http://localhost:5173`.

### Build for production

```bash
bunx vite build
```

### Check types

```bash
bunx svelte-check --tsconfig ./tsconfig.json
```

### Run tests

```bash
bunx vitest
```

## API Integration

The dashboard proxies API requests to the Go backend (default: `http://localhost:8080`). Configure the proxy target in `vite.config.ts` under `server.proxy`.

**Health Check Endpoint**: The home page (`/`) automatically calls `/api/health` on load to verify backend connectivity.

## Styling

- **Tailwind CSS**: Configured via `tailwindcss.config.ts`
- **Glasmorphism Theme**: Applied in `src/routes/+page.svelte` as an example

## Code Quality

### ESLint

The app uses a **legacy `.eslintrc.json` config** (compatible with monorepo's ESLint 8.46.0) that extends the root monorepo rules:

```bash
bunx eslint .
```

Rules align with the root `.eslintrc.json` (Google style guide, TypeScript safety, naming conventions). Svelte-specific rules are included via `eslint-plugin-svelte`.

**Note**: `svelte.config.js` and `vite.config.mjs` are excluded from linting (see `.eslintignore`).

### Prettier

The app shares the root monorepo Prettier config (`.prettierrc`):

```bash
bunx prettier --check .
bunx prettier --write .
```

**Config**: `useTabs: false`, `tabWidth: 2`, `trailingComma: "all"`, `printWidth: 120`. Svelte-specific plugins (`prettier-plugin-svelte`, `prettier-plugin-tailwindcss`) are included for proper `.svelte` file formatting.

## Nx Integration

The `project.json` file enables Nx task tracking:

```bash
# Build via Nx
bunx nx build wealth-management-dashboard

# Serve via Nx
bunx nx serve wealth-management-dashboard

# Check via Nx
bunx nx check wealth-management-dashboard
```

## Status

**Sprint 1**: ✅ Scaffolded, ✅ FSD structure created, ✅ API health check implemented, ✅ Glassmorphism UI theme integrated.
