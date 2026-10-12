# Installation and Lifecycle Flow

First run of `apps/wealth-management`. Commands run from the repo root.

## 1. Install

```sh
pnpm install
```

## 2. Configure

1. Create `.env.local` at the repo root with `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` and `GOOGLE_SHEETS_ID` (see [setup README §3](../README.md#3-environment-variables)).
2. Mint the refresh token. This prints a consent URL, asks for the redirected URL, and writes `GOOGLE_REFRESH_TOKEN` into `.env.local`:

   ```sh
   pnpm run auth:sheets
   ```

3. Verify the token and sheet access:

   ```sh
   pnpx nx run wealth-management:verify-auth
   ```

4. Optionally add cache, AI, search and market variables.

## 3. Run

| Mode      | Command                               | Result                                                                           |
| --------- | ------------------------------------- | -------------------------------------------------------------------------------- |
| Dev       | `pnpx nx run wealth-management:serve` | Engine with live reload on `:8080` plus the Vite dev server with `/api` proxied. |
| Prod-like | `pnpx nx run wealth-management:start` | Builds the UI, then runs the engine, which serves the UI at `/ui` (one origin).  |

Then check `GET /api/health` (per-dependency status) and open `/api/docs` for the API reference.

## 4. Operations

- **Refresh data**: sheet reads are cached; `POST /api/sync` invalidates every sheet-derived key (WM-OPS-3).
- **Token expired** (`invalid_grant`): rerun `pnpm run auth:sheets`, then `verify-auth`.
- **Tests**: `pnpx nx run wealth-management:test` and `pnpx nx run wealth-management:test-ui`. Live tests: [setup README §6](../README.md#6-live-e2e-tests-disposable-sheet-copy).
