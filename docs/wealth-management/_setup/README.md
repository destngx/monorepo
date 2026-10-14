# Wealth Management: Setup and Configuration

How to configure and run `apps/wealth-management` (Go engine + SvelteKit UI, one Nx project). Scope and feature status live in [REQUIREMENTS.md](../REQUIREMENTS.md); this page covers only setup.

| Page                                                                                                 | Covers                                              |
| ---------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| [Infrastructure and prerequisites](./1-Environment-Requirements/Infrastructure_and_Prerequisites.md) | Toolchain and external services                     |
| [Variable definitions and secrets](./2-System-Configuration/Variable_Definitions_and_Secrets.md)     | Pointer to the settings tables below, secrets rules |
| [Installation and lifecycle flow](./3-Operations-and-Running/Installation_and_Lifecycle_Flow.md)     | First run, OAuth, verification, token recovery      |

---

## 1. Layout

- Go engine (Fiber, hexagonal: `domain/`, `port/`, `service/`, `adapter/`, `cmd/`) at `apps/wealth-management/`, entry point `main.go`.
- SvelteKit UI in `apps/wealth-management/ui/`, built with `adapter-static` (base `/ui`) into `ui/build/` and served by the engine at `/ui` (`/` redirects there). If `ui/build/` is missing the engine still starts and logs a hint.
- API under `/api`. OpenAPI spec at `/api/openapi.json`, Swagger UI at `/api/docs`, dependency status at `/api/health`.
- MCP server over stdio runs in the same process. Stdout carries JSON-RPC; logs go to stderr.

## 2. Settings (no env file)

There is no env file. The engine reads its settings from a SQLite file, edited on the **Settings** page of the UI (Engine settings), so the engine starts with no configuration at all and the page sets it up.

- **Location**: `settings.db` in the user config directory (macOS `~/Library/Application Support/wealth-management/`, Linux `$XDG_CONFIG_HOME/wealth-management/`), or `WM_SETTINGS_PATH`. The directory is `0700` and the file `0600`: secrets are stored as plain text, protected by the file mode.
- **Secrets are write-only**: `GET /api/settings` reports whether a secret is set, never its value. The page can replace or clear a secret but never shows it.
- **Applying changes**: saved settings take effect after a restart. The page's **Restart engine** button (`POST /api/settings/restart`) shuts the server down gracefully and re-executes the same binary with the same arguments, pid and port. The page waits until the engine is back.
- **Validation**: `PATCH /api/settings` rejects unknown keys, multi-line values, non-positive integers, non-http(s) URLs, unknown choices, only one of the two Upstash settings, and `CAPTURE_MAX_ITEM_MB` above `CAPTURE_MAX_BATCH_MB`. Writes need the API token like every other write (or loopback when `API_TOKEN` is unset).
- **Bad stored values never stop the engine**: an invalid card programs file or capture limit falls back to the defaults with an error log, so the Settings page stays reachable to fix it.
- **Process environment is ignored for settings**: setting `GOOGLE_SHEETS_ID` and friends in the shell has no effect. Only the bootstrap variables in section 3.1 are read from the environment.
- **Migrating an old `.env.local`**: `pnpx nx run wealth-management:import-settings` copies the known keys from `../../.env.local` and `apps/wealth-management/.env.local` (the app file wins) into the store, or from the files passed after `--`. It prints key names only, never values. Delete the env files afterwards.

## 3. Settings reference

Every setting, by the group it appears under on the Settings page. Keys keep the former variable names. Empty means unset.

### 3.1 Bootstrap (process environment)

The only values read from the environment: the engine needs them before it can open the settings store, or they belong to the process rather than the engine.

| Variable            | Default                                    | Meaning                                                                                                                                     |
| ------------------- | ------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------- |
| `HOST`              | `127.0.0.1`                                | Bind address.                                                                                                                               |
| `PORT`              | `8080`                                     | Listen port (1-65535; anything else aborts startup). The UI dev proxy targets `localhost:8080`.                                             |
| `API_TOKEN`         | empty                                      | Bearer token required on write endpoints. When empty, writes are allowed only if `HOST` is a loopback address, and refused (403) otherwise. |
| `WM_SETTINGS_PATH`  | user config directory                      | Settings store file (section 2).                                                                                                            |
| `CACHE_SQLITE_PATH` | `<system temp>/wealth-management/cache.db` | SQLite cache file, used when Upstash is not configured. Several engines (dev server, e2e runs) can share the file.                          |
| `LOG_COLOR`         | enabled                                    | Set to `false` to disable coloured logs.                                                                                                    |
| `WM_TEST_MODE`      | `false`                                    | Test mode for live e2e runs: startup aborts unless the spreadsheet title starts with `wm-e2e-` (WM-TEST-2). Set by the harness only.        |

### 3.2 General

| Setting              | Default       | Meaning                                                                                                                                                           |
| -------------------- | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `LOG_LEVEL`          | `info`        | `debug`, `info`, `warn` or `error`.                                                                                                                               |
| `CARD_PROGRAMS_FILE` | embedded file | Path to a JSON file replacing the embedded credit card programs (`adapter/config/cards.default.json`, WM-CARD-3). An invalid file falls back to the embedded one. |

### 3.3 Google Sheets (system of record)

All four of `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REFRESH_TOKEN`, `GOOGLE_SHEETS_ID` are required for the sheet-backed routes. If any is missing the engine still starts, logs which ones, disables the sheet and AI routes, and `/api/health` reports `sheets` as unconfigured (503).

| Setting                   | Default         | Meaning                                                                                                                                                                                             |
| ------------------------- | --------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOOGLE_CLIENT_ID`        | none            | OAuth client id (Google Cloud project with the Sheets API enabled).                                                                                                                                 |
| `GOOGLE_CLIENT_SECRET`    | none            | OAuth client secret.                                                                                                                                                                                |
| `GOOGLE_REFRESH_TOKEN`    | none            | Long-lived refresh token, saved by **Connect Google account** (section 3.8). Scope: `spreadsheets` only.                                                                                            |
| `GOOGLE_SHEETS_ID`        | none            | Spreadsheet id from the sheet URL.                                                                                                                                                                  |
| `GOOGLE_REDIRECT_URL`     | UI callback     | Where Google returns after sign-in. Empty uses `/api/auth/google/callback` on the loopback address the UI is open on (section 3.8). A Web OAuth client must list it exactly.                        |
| `BUDGET_SHEET_TEMPLATE`   | `Budget_{year}` | Budget tab name; `{year}` is replaced with the requested year (WM-DATA-2).                                                                                                                          |
| `SHEETS_READS_PER_MINUTE` | `60`            | Pace of Sheets read requests (WM-NFR-3); bursts up to a sixth of it, then excess reads wait their turn instead of failing. Must be a positive integer, or the sheet routes are disabled at startup. |

### 3.4 Cache (Upstash Redis REST or local SQLite)

Optional. With both the Upstash URL and token set, the cache is Upstash. With neither, it is a local SQLite file at `CACHE_SQLITE_PATH` (section 3.1); the default file lives in the system temp directory (macOS `$TMPDIR`, Linux `/tmp`), is readable only by the user, and may be lost on reboot, which only costs a cold cache. The Settings page refuses only one of the two Upstash settings.

| Setting                    | Default | Meaning                                                                                                          |
| -------------------------- | ------- | ---------------------------------------------------------------------------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | none    | Upstash REST endpoint.                                                                                           |
| `UPSTASH_REDIS_REST_TOKEN` | none    | Upstash REST token.                                                                                              |
| `CACHE_KEY_PREFIX`         | `wm:`   | Prefix on every cache key, so environments and test runs never share keys (WM-OPS-2b). Applies to both backends. |

### 3.5 AI (OpenAI-compatible provider)

AI routes are disabled, and `GET /api/ai/status` says why, until `AI_BASE_URL` and a model for every use case (normally via `AI_MODEL`) are set. They also need the Google Sheet, because prompts come from its `Prompts` tab. See REQUIREMENTS.md §10.1.

`GET /api/ai/usage` (Settings, AI card) shows the OpenAI quota by calling the ai-gateway `GET {AI_BASE_URL}/usage` with `X-AI-Provider: openai`. It needs only `AI_BASE_URL` (and `AI_API_KEY` if the gateway requires one); it answers 503 UNAVAILABLE when the gateway has no OpenAI provider configured or cannot reach it.

| Setting                | Default    | Meaning                                                                                                                   |
| ---------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------- |
| `AI_BASE_URL`          | none       | OpenAI-compatible base URL, e.g. `http://127.0.0.1:8081/v1` for ai-gateway.                                               |
| `AI_API_KEY`           | empty      | Bearer key for the provider (optional for a local gateway).                                                               |
| `AI_PROVIDER`          | empty      | Sent as `X-AI-Provider` on chat calls (ai-gateway routing). The usage check always sends `openai`.                        |
| `AI_MODEL`             | none       | Default model for every use case.                                                                                         |
| `AI_MODEL_INSIGHT`     | `AI_MODEL` | Model for reviews, briefings and insights.                                                                                |
| `AI_MODEL_SUGGEST`     | `AI_MODEL` | Model for category suggestion.                                                                                            |
| `AI_MODEL_VISION`      | `AI_MODEL` | Image-capable model for Smart Capture.                                                                                    |
| `AI_MODEL_CHAT`        | `AI_MODEL` | Model for chat.                                                                                                           |
| `AI_FALLBACK_MODEL`    | empty      | Retried once when the primary model fails or returns invalid output.                                                      |
| `AI_TIMEOUT_SECONDS`   | `90`       | Per-call timeout; must be a positive integer.                                                                             |
| `AI_PII_NAMES`         | empty      | Comma-separated names masked as `[name]` before text reaches a model.                                                     |
| `CAPTURE_MAX_ITEMS`    | `20`       | Smart Capture: materials per batch.                                                                                       |
| `CAPTURE_MAX_ITEM_MB`  | `5`        | Smart Capture: size per material. Must not exceed `CAPTURE_MAX_BATCH_MB`.                                                 |
| `CAPTURE_MAX_BATCH_MB` | `20`       | Smart Capture: size per batch. HTTP body and MCP request limits grow to fit it. Invalid values fall back to the defaults. |

### 3.6 Web search

Optional. Tried in order Tavily, Exa, then DuckDuckGo (no key).

| Setting          | Default | Meaning         |
| ---------------- | ------- | --------------- |
| `TAVILY_API_KEY` | empty   | Tavily API key. |
| `EXA_API_KEY`    | empty   | Exa API key.    |

### 3.7 Market data server

The engine does not call market sources (fmarket, vnstock, Yahoo Finance, CoinGecko, currency-api) itself. It calls one service, [market-data-server](../../market-data-server/MARKET_PLATFORM.md), which owns the source chains, caching and last-good copies (see [Market provider capabilities](../_technical/Market_Provider_Capabilities.md)). Source base URLs, the CoinGecko key, `MARKET_CHAINS` and `MARKET_DB_PATH` are set on the server in `apps/market-data-server/.env.local`, not in the engine settings.

| Setting                  | Default                 | Meaning                                                                            |
| ------------------------ | ----------------------- | ---------------------------------------------------------------------------------- |
| `MARKET_DATA_SERVER_URL` | `http://localhost:8000` | Base URL of `apps/market-data-server`, which serves all market data (WM-MKT-1..7). |

### 3.8 First run

1. Start the engine (`pnpx nx run wealth-management:start`) and open `/ui/settings`.
2. Under Google Sheets, set `GOOGLE_SHEETS_ID`, `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET`, and save.
3. Click **Connect Google account** on the Google Sheets card. The tab goes to Google's consent page; allow access and the engine saves `GOOGLE_REFRESH_TOKEN` and brings you back to Settings, which confirms it and shows **Restart to apply**. The sign-in expires after 10 minutes and works once.
4. Click **Restart engine**. Add the cache, AI and search settings as needed, then restart again.

Google returns the browser to `GET /api/auth/google/callback` on the address the UI is open on, when that is a loopback address: `http://127.0.0.1:5173/api/auth/google/callback` with the Vite dev server (it forwards `/api` to the engine), `http://127.0.0.1:8080/api/auth/google/callback` when the engine serves the UI. From any other host it uses the engine's `http://127.0.0.1:<PORT>` callback, because Google accepts plain-http redirects on loopback only. `GOOGLE_REDIRECT_URL` overrides all of this. The dialog shows the address in use. The callback needs no API token: a one-time state and a PKCE verifier held by the engine protect it, and the request log redacts `code` and `state`. The OAuth client must allow the redirect:

- A **Desktop** client accepts any loopback port, so every address above works as is.
- A **Web** client must list each exact redirect URI it is used with, e.g. both the `5173` (dev) and `8080` (engine) callbacks. A mismatch is Google's `Error 400: redirect_uri_mismatch`.
- After consent the callback redirects (303) to `/ui/settings?google=connected&purpose=sheets|test`, or `google=error&reason=denied|expired|rejected|no_refresh_token|invalid|failed`. The page turns the fixed reason into a message, so a crafted link cannot show arbitrary text, then removes the query.
- When Google returns to another origin than the page (the UI opened from another host, or `GOOGLE_REDIRECT_URL` set elsewhere), the sign-in opens in a new tab instead. If that tab ends on a page that cannot be reached, paste its address into the dialog; it is sent to `POST /api/auth/google/exchange`.

Re-authorise the same way when the token expires (`invalid_grant`). **Connect test account** on the Testing card mints `GOOGLE_TEST_REFRESH_TOKEN` (section 6).

## 4. Nx targets

Run from the repo root with `pnpx nx run wealth-management:<target>`. pnpm only; never bun, npm or yarn.

| Target            | What it does                                                                                         |
| ----------------- | ---------------------------------------------------------------------------------------------------- |
| `serve`           | Dev: runs `serve-api` and `serve-ui` in parallel.                                                    |
| `serve-api`       | Engine with live reload (`air`, `.air.toml`), port 8080 by default.                                  |
| `serve-ui`        | Vite dev server for the UI (default port 5173, `/api` proxied to `localhost:8080`).                  |
| `start`           | Prod-like: builds the UI (`build-ui:production`), then `go run .`; UI served by the engine at `/ui`. |
| `build`           | Go build of the engine.                                                                              |
| `build-ui`        | `vite build` into `ui/build/`.                                                                       |
| `test`            | Go unit and contract tests (`go test ./...`).                                                        |
| `test-ui`         | Vitest for the UI.                                                                                   |
| `check-ui`        | `svelte-check` type check.                                                                           |
| `lint`            | Go lint.                                                                                             |
| `lint-ui`         | ESLint for the UI.                                                                                   |
| `test-e2e`        | Live tests (`RUN_LIVE_API_TESTS=true`, `TestE2E*`). See section 6.                                   |
| `verify-auth`     | Checks the OAuth credentials and sheet access (`cmd/verify_auth`).                                   |
| `import-settings` | Copies an old `.env.local` into the settings store (`cmd/import_settings`).                          |

## 5. CLIs

The CLIs read and write the same settings store as the engine (`WM_SETTINGS_PATH` or the user config directory). Restart the engine after they change it.

- **`verify-auth`** (`cmd/verify_auth`): reads `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REFRESH_TOKEN` and `GOOGLE_SHEETS_ID`, refreshes an access token, then opens the spreadsheet. `invalid_grant` means the refresh token expired or was revoked (a Google Cloud project in "Testing" mode expires tokens after 7 days): use **Connect Google account** again (section 3.8).
- **`import-settings`** (`cmd/import_settings`): see section 2. Values are validated like the Settings page; bootstrap and unknown keys are listed by name and skipped.

## 6. Live E2E tests (disposable test sheet)

Live tests never touch the production sheet ([REQUIREMENTS.md §14.3](../REQUIREMENTS.md#143-disposable-sheet-copy-never-test-against-the-real-sheet)). Each run uploads a local `.xlsx` export of the workbook to Drive and converts it into a new Google Sheet. It starts the engine against that sheet and deletes it afterwards. The harness never opens the production sheet.

**One-time setup**:

1. Mint the test token: click **Connect test account** on the Testing card of the Settings page. It asks for the `spreadsheets` and `drive.file` scopes and saves `GOOGLE_TEST_REFRESH_TOKEN`. The production token keeps the `spreadsheets` scope only.

2. Export the workbook as `.xlsx` (File > Download > Microsoft Excel). Keep it **outside the repo**, because it holds real financial data.
3. Point the harness at it: set `E2E_SOURCE_XLSX` under Testing on the Settings page (for example `~/projects/Money Manager.xlsx`), or export it in the shell, which wins.

Re-export the workbook whenever the sheet's structure changes.

**Run**:

```sh
pnpx nx run wealth-management:test-e2e
# or, from apps/wealth-management:
RUN_LIVE_API_TESTS=true go test ./e2e/ -run '^TestE2E' -count=1 -v
```

Without `RUN_LIVE_API_TESTS=true` the suite is skipped, so `go test ./...` stays offline.

| Name                        | Meaning                                                                                                                  |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `RUN_LIVE_API_TESTS`        | Environment: `true` opts in to the live suite.                                                                           |
| `E2E_SOURCE_XLSX`           | Setting or environment: path to the `.xlsx` export used as test data; a leading `~/` is expanded.                        |
| `GOOGLE_TEST_REFRESH_TOKEN` | Setting: test-only token from **Connect test account**. Scopes: `spreadsheets` and `drive.file` (only files it created). |
| `KEEP_TEST_SHEET`           | Environment: `true` keeps the test sheet after the run for debugging; its name is logged.                                |

`GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` come from the settings store, which a run only reads. `GOOGLE_SHEETS_ID` is optional here. When it is set, it is used only to refuse the production sheet.

What a run does (`apps/wealth-management/e2e/harness_test.go`, `internal/e2esheet`):

1. **Janitor**: deletes leftover `wm-e2e-*` test sheets older than 24 h. Because of the `drive.file` scope, it can only see sheets that the harness itself created.
2. **Upload**: converts the workbook into a sheet named `wm-e2e-<runId>-<UTC timestamp>`. Tabs, formulas, `Prompts` and `Knowledge` come along. It aborts if the new id equals `GOOGLE_SHEETS_ID` or the name lacks the prefix.
3. **Engine**: builds and starts the engine with a settings store of its own in the run's work directory, seeded from the user's settings (without the Testing group):
   - the store points at the test sheet, uses the test token as `GOOGLE_REFRESH_TOKEN` and sets `CACHE_KEY_PREFIX=wm-test:<runId>:`;
   - the environment holds only bootstrap variables: `WM_SETTINGS_PATH`, `CACHE_SQLITE_PATH` in the work directory, `WM_TEST_MODE=true`, a random `API_TOKEN` and a free port on 127.0.0.1.

   In test mode the engine refuses to start unless the sheet title has the `wm-e2e-` prefix.

4. **Tests**, run serially on the one sheet:
   - Every sheet read route in the OpenAPI spec parses.
   - Structural checks.
   - Round-trips for accounts, transactions (including the stale ETag 409), idempotent batches, budget limits and Smart Capture rules.
   - Writes without the token get 401.
   - A market smoke test.

   Assertions are relative or structural, because the workbook holds real data that changes with each export.

5. **Teardown**: invalidates the run's cache keys and stops the engine. It deletes the test sheet unless `KEEP_TEST_SHEET=true`; before deleting, it re-checks the name prefix. The engine log is in `apps/wealth-management/tmp/e2e-<runId>/engine.log` and is kept when the run fails.
