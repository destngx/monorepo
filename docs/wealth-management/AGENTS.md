# Wealth Management: Engineering & Agent Standards

This document defines the core standards for the Wealth Management platform, covering the Go backend and SvelteKit frontend.

**Scope and status of features**: [`REQUIREMENTS.md`](./REQUIREMENTS.md) is the source of truth.

## 📦 Layout

One Nx project, `wealth-management`, at `apps/wealth-management/`:

- Go engine (Fiber) at the project root, API under `/api`, MCP server over stdio (stdout is reserved for JSON-RPC; logs go to stderr).
- SvelteKit UI in `ui/`, built with `adapter-static` (base `/ui`) and served by the engine at `/ui`. One process, one origin in production.
- Dev: `pnpx nx run wealth-management:serve` (air + vite with `/api` proxied to `:9090`). Prod-like: `pnpx nx run wealth-management:start` (builds the UI, then runs the engine).
- Settings: no env file. Integration settings live in a SQLite store edited on the UI Settings page (`GET/PATCH /api/settings`, applied by `POST /api/settings/restart`); only bootstrap values (`HOST`, `PORT`, `WM_SETTINGS_PATH`, `WM_CAPTURE_PATH`, `CACHE_SQLITE_PATH`, `WM_TEST_WORKBOOK`, `LOG_COLOR`) come from the environment. Registry: `domain/settings.go`. The API token (WM-SEC-1) is stored there too but set on its own endpoint (`/api/settings/api-token`), applied at once.
- Google OAuth: **Connect Google account** in Settings (`/api/auth/google/*`, saves to the settings store); check with `pnpx nx run wealth-management:verify-auth`.

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

### ⏳ Loading and feedback (WM-UI-16..19)

Pages never wait for the API. A load function returns un-awaited promises (SvelteKit streams them), the page renders its shell (header, filters, layout) at once, and each slow section shows its own skeleton, then its data or its own error with a Retry.

- **Load**: wrap every request in `defer(() => api.x({ fetch }))` (`$lib/api/load`). It returns `Streamed<T>` = `Promise<Settled<T>>` and never rejects: a failure resolves to `{ ok: false, error, failure }` with `failure.kind` = `auth` | `not-configured` | `unavailable` | `unreachable` | `other`. A Google Sheets credential failure (expired token, Sheets not configured) also raises the layout's re-auth alert (SheetsAuthAlert), so the page explains the way out once while each section shows its own short error. Await (with `loadOrError`, which goes to `+error.svelte`) only data the shell cannot render without, and say why in a comment; values from the URL (month, code) need no request.
- **Render**: `<Deferred data={data.x} label="what loads">` with a `children(value)` snippet; optional `skeleton` and `error(failure, retry)` snippets, `retry` (default `invalidateAll`) and `scope` (drop the old value when it changes, e.g. the month). Put spacing (`class="mt-4"`) on `<Deferred>`, not on the card inside it.
- **Values in the script**: `const x = track(() => data.x, { scope })` (`$lib/api/deferred.svelte`) gives `x.value`, `x.loading`, `x.refreshing`, `x.failure` for headers, buttons and modals that need the data outside a section.
- **Refresh**: a newer promise (sync, `invalidate`, retry, or a page swapping in `defer(...)` after a forced refresh) keeps the shown value with a small Refreshing badge; a failed refresh keeps it under an error strip. Skeletons only show before the first value.
- **Skeletons** (`$lib/components/skeleton`): `SkeletonText`, `SkeletonStat`, `SkeletonCard` (`title`, `lines`, `chart`, `rows`), `SkeletonTable`, `SkeletonChart`, `SkeletonHeadline`. Each is a busy `role="status"` with "Loading {label}…" for screen readers and pulses only without reduced motion.
- **App loading**: until the root layout first renders, `app.html` shows an outline of the app shell (sidebar at its saved width, header, page cards, mobile tab bar) with the page-load bar; `endBootScreen` removes it. Keep its sizes in step with `routes/+layout.svelte`. Page changes show `NavProgress` (`$lib/components`): nothing for navigations under 150ms, then a bar that creeps towards the end and fills and fades when the page arrives.
- **Actions**: `<Button>` (`$lib/components/Button.svelte`, variants `primary` | `ghost` | `danger`, sizes `md` | `sm` | `icon`): an `onclick` that returns a promise makes it busy (spinner, `aria-busy`, repeat clicks ignored) and toasts `success` or the error (`errorContext` prefix); `busy` forces the state; icon-only buttons take `label` (aria-label and tooltip). Outside a button, `action(fn, { success, errorContext })` (`$lib/state/action.svelte`) gives `busy` and `run`.
- **Feedback styles** (`routes/layout.css`): `.btn-*` hover, press, focus ring, disabled and busy states; `.field` hover and focus ring; `.link` for text links; `.interactive` for a clickable card (lift on hover, sink on press); `.row-link` for a clickable row; `.skeleton`; `.icon-btn`. Every page also gets, from base styles, pointer cursors, a 1px press on buttons, table row hover, bare link styling and reduced-motion handling.

```ts
// routes/example/+page.ts
export const load: PageLoad = ({ fetch }) => ({
  report: defer(() => api.report({ fetch })),
});
```

```svelte
<!-- routes/example/+page.svelte -->
<PageHeader title="Example">
  {#snippet actions()}<Button icon="refresh" success="Refreshed." onclick={() => invalidateAll()}>Refresh</Button
    >{/snippet}
</PageHeader>
<Deferred data={data.report} label="the report" class="mt-4">
  {#snippet children(report)}<ReportCard {report} />{/snippet}
  {#snippet skeleton()}<SkeletonCard title="Report" lines={3} chart={180} />{/snippet}
</Deferred>
```

A page test renders the page with never-settling promises (`new Promise(() => {})`) and checks the shell and the "Loading …" statuses, then settles one part with `settled(value)` or `defer(Promise.reject(...))` (see `routes/budget/page.test.ts`).

## 🧪 Testing & Workflow

- **Standard**: TDD + BDD.
- **Tooling**: pnpm only (`pnpm run`, `pnpx nx`).
- **Backend Test**: `pnpx nx run wealth-management:test`
- **Frontend Test**: `pnpx nx run wealth-management:test-ui` (type check: `check-ui`, lint: `lint-ui`)
- **Live tests**: only against a local `.xlsx` export served in memory by the local workbook adapter, never the production sheet (REQUIREMENTS.md §14).
- **Security**: Never print or log setting values; secrets are write-only in the settings API (never returned).
