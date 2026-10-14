# Installation and Lifecycle Flow

First run of `apps/wealth-management`. Commands run from the repo root.

## 1. Install

```sh
pnpm install
```

## 2. Configure

There is no env file ([setup README §2](../README.md#2-settings-no-env-file)). Start the engine (section 3) and open `/ui/settings`.

1. Under Engine settings > Google Sheets, set `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` and `GOOGLE_SHEETS_ID`, and save. An existing `.env.local` can be imported instead with `pnpx nx run wealth-management:import-settings`.
2. Click **Connect Google account** on the Google Sheets card and allow access in the Google tab. The engine saves `GOOGLE_REFRESH_TOKEN` to the settings store ([setup README §3.8](../README.md#38-first-run) covers the redirect the OAuth client must allow).

3. Verify the token and sheet access:

   ```sh
   pnpx nx run wealth-management:verify-auth
   ```

4. Optionally add cache, AI, search and market settings, then click **Restart engine** on the Settings page to apply them.

## 3. Run

| Mode      | Command                               | Result                                                                           |
| --------- | ------------------------------------- | -------------------------------------------------------------------------------- |
| Dev       | `pnpx nx run wealth-management:serve` | Engine with live reload on `:9090` plus the Vite dev server with `/api` proxied. |
| Prod-like | `pnpx nx run wealth-management:start` | Builds the UI, then runs the engine, which serves the UI at `/ui` (one origin).  |

Then check `GET /api/health` (per-dependency status) and open `/api/docs` for the API reference.

## 4. Operations

- **Refresh data**: sheet reads are cached; `POST /api/sync` invalidates every sheet-derived key (WM-OPS-3).
- **Token expired** (`invalid_grant`): use **Connect Google account** in Settings again, then `verify-auth`, then restart the engine from Settings.
- **Tests**: `pnpx nx run wealth-management:test` and `pnpx nx run wealth-management:test-ui`. Live tests: [setup README §6](../README.md#6-live-e2e-tests-local-workbook).
