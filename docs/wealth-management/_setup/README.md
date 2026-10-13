# Wealth Management: Setup and Configuration

How to configure and run `apps/wealth-management` (Go engine + SvelteKit UI, one Nx project). Scope and feature status live in [REQUIREMENTS.md](../REQUIREMENTS.md); this page covers only setup.

| Page                                                                                                 | Covers                                         |
| ---------------------------------------------------------------------------------------------------- | ---------------------------------------------- |
| [Infrastructure and prerequisites](./1-Environment-Requirements/Infrastructure_and_Prerequisites.md) | Toolchain and external services                |
| [Variable definitions and secrets](./2-System-Configuration/Variable_Definitions_and_Secrets.md)     | Pointer to the env tables below, secrets rules |
| [Installation and lifecycle flow](./3-Operations-and-Running/Installation_and_Lifecycle_Flow.md)     | First run, OAuth, verification, token recovery |

---

## 1. Layout

- Go engine (Fiber, hexagonal: `domain/`, `port/`, `service/`, `adapter/`, `cmd/`) at `apps/wealth-management/`, entry point `main.go`.
- SvelteKit UI in `apps/wealth-management/ui/`, built with `adapter-static` (base `/ui`) into `ui/build/` and served by the engine at `/ui` (`/` redirects there). If `ui/build/` is missing the engine still starts and logs a hint.
- API under `/api`. OpenAPI spec at `/api/openapi.json`, Swagger UI at `/api/docs`, dependency status at `/api/health`.
- MCP server over stdio runs in the same process. Stdout carries JSON-RPC; logs go to stderr.

## 2. Environment file

`config.Load` (in `adapter/config/config.go`) reads two files, relative to the engine working directory `apps/wealth-management`:

1. `.env.local` at the repo root (`../../.env.local`).
2. `apps/wealth-management/.env.local`.

Both are optional and loaded with override semantics: the app-level file wins over the root file, and both win over variables already set in the shell. `.env.local` is gitignored and holds runtime input only: never commit, print or log it.

`WM_ENV_FILE` (read from the shell) overrides this: `none` skips both files, any other value loads only that path. The live e2e harness starts the engine with `WM_ENV_FILE=none` so its own variables are not overridden.

## 3. Environment variables

Every variable the engine and its CLIs read. Empty means unset.

### 3.1 Server and logging

| Variable             | Default       | Meaning                                                                                                                                        |
| -------------------- | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `HOST`               | `127.0.0.1`   | Bind address.                                                                                                                                  |
| `PORT`               | `8080`        | Listen port (1-65535; anything else aborts startup). The UI dev proxy targets `localhost:8080`.                                                |
| `API_TOKEN`          | empty         | Bearer token required on write endpoints. When empty, writes are allowed only if `HOST` is a loopback address, and refused (403) otherwise.    |
| `LOG_LEVEL`          | `info`        | Log level. Read after `.env.local` is loaded.                                                                                                  |
| `LOG_COLOR`          | enabled       | Set to `false` to disable coloured logs.                                                                                                       |
| `WM_TEST_MODE`       | `false`       | Test mode for live e2e runs: startup aborts unless the spreadsheet title starts with `wm-e2e-` (WM-TEST-2). Set by the harness only.           |
| `CARD_PROGRAMS_FILE` | embedded file | Path to a JSON file replacing the embedded credit card programs (`adapter/config/cards.default.json`, WM-CARD-3). Invalid file aborts startup. |

### 3.2 Google Sheets (system of record)

All four of `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REFRESH_TOKEN`, `GOOGLE_SHEETS_ID` are required for the sheet-backed routes. If any is missing the engine still starts, logs which ones, disables the sheet and AI routes, and `/api/health` reports `sheets` as unconfigured (503).

| Variable                  | Default                 | Meaning                                                                                                                                                                                             |
| ------------------------- | ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOOGLE_CLIENT_ID`        | none                    | OAuth client id (Google Cloud project with the Sheets API enabled).                                                                                                                                 |
| `GOOGLE_CLIENT_SECRET`    | none                    | OAuth client secret.                                                                                                                                                                                |
| `GOOGLE_REFRESH_TOKEN`    | none                    | Long-lived refresh token, written by `pnpm run auth:sheets`. Scope: `spreadsheets` only.                                                                                                            |
| `GOOGLE_SHEETS_ID`        | none                    | Spreadsheet id from the sheet URL.                                                                                                                                                                  |
| `GOOGLE_REDIRECT_URL`     | `http://127.0.0.1:3000` | OAuth redirect URL; must match the OAuth client. Used by the engine and `auth-sheets`.                                                                                                              |
| `BUDGET_SHEET_TEMPLATE`   | `Budget_{year}`         | Budget tab name; `{year}` is replaced with the requested year (WM-DATA-2).                                                                                                                          |
| `SHEETS_READS_PER_MINUTE` | `60`                    | Pace of Sheets read requests (WM-NFR-3); bursts up to a sixth of it, then excess reads wait their turn instead of failing. Must be a positive integer, or the sheet routes are disabled at startup. |

### 3.3 Cache (Upstash Redis REST)

Optional. Without both URL and token the cache is disabled and reads go straight to the sheet.

| Variable                   | Default | Meaning                                                                                |
| -------------------------- | ------- | -------------------------------------------------------------------------------------- |
| `UPSTASH_REDIS_REST_URL`   | none    | Upstash REST endpoint.                                                                 |
| `UPSTASH_REDIS_REST_TOKEN` | none    | Upstash REST token.                                                                    |
| `CACHE_KEY_PREFIX`         | `wm:`   | Prefix on every cache key, so environments and test runs never share keys (WM-OPS-2b). |

### 3.4 AI (OpenAI-compatible provider)

AI routes are disabled, and `GET /api/ai/status` says why, until `AI_BASE_URL` and a model for every use case (normally via `AI_MODEL`) are set. They also need the Google Sheet, because prompts come from its `Prompts` tab. See REQUIREMENTS.md §10.1.

`GET /api/ai/usage` (Settings, AI card) shows the OpenAI quota by calling the ai-gateway `GET {AI_BASE_URL}/usage` with `X-AI-Provider: openai`. It needs only `AI_BASE_URL` (and `AI_API_KEY` if the gateway requires one); it answers 503 UNAVAILABLE when the gateway has no OpenAI provider configured or cannot reach it.

| Variable               | Default    | Meaning                                                                                                       |
| ---------------------- | ---------- | ------------------------------------------------------------------------------------------------------------- |
| `AI_BASE_URL`          | none       | OpenAI-compatible base URL, e.g. `http://127.0.0.1:8081/v1` for ai-gateway.                                   |
| `AI_API_KEY`           | empty      | Bearer key for the provider (optional for a local gateway).                                                   |
| `AI_PROVIDER`          | empty      | Sent as `X-AI-Provider` on chat calls (ai-gateway routing). The usage check always sends `openai`.            |
| `AI_MODEL`             | none       | Default model for every use case.                                                                             |
| `AI_MODEL_INSIGHT`     | `AI_MODEL` | Model for reviews, briefings and insights.                                                                    |
| `AI_MODEL_SUGGEST`     | `AI_MODEL` | Model for category suggestion.                                                                                |
| `AI_MODEL_VISION`      | `AI_MODEL` | Image-capable model for Smart Capture.                                                                        |
| `AI_MODEL_CHAT`        | `AI_MODEL` | Model for chat.                                                                                               |
| `AI_FALLBACK_MODEL`    | empty      | Retried once when the primary model fails or returns invalid output.                                          |
| `AI_TIMEOUT_SECONDS`   | `90`       | Per-call timeout; a non-integer value falls back to the default.                                              |
| `AI_PII_NAMES`         | empty      | Comma-separated names masked as `[name]` before text reaches a model.                                         |
| `CAPTURE_MAX_ITEMS`    | `20`       | Smart Capture: materials per batch.                                                                           |
| `CAPTURE_MAX_ITEM_MB`  | `5`        | Smart Capture: size per material. Must not exceed `CAPTURE_MAX_BATCH_MB`.                                     |
| `CAPTURE_MAX_BATCH_MB` | `20`       | Smart Capture: size per batch. HTTP body and MCP request limits grow to fit it. Invalid values abort startup. |

### 3.5 Web search

Optional. Tried in order Tavily, Exa, then DuckDuckGo (no key).

| Variable         | Default | Meaning         |
| ---------------- | ------- | --------------- |
| `TAVILY_API_KEY` | empty   | Tavily API key. |
| `EXA_API_KEY`    | empty   | Exa API key.    |

### 3.6 Market data providers

All optional. A provider that fails to build is dropped and its routing chains fall through to the next provider (see [Market provider capabilities](../_technical/Market_Provider_Capabilities.md)).

| Variable                 | Default                                                         | Meaning                                                |
| ------------------------ | --------------------------------------------------------------- | ------------------------------------------------------ |
| `FMARKET_BASE_URL`       | `https://api.fmarket.vn`                                        | fmarket API (funds, gold, USD/VND, bank rates).        |
| `VNSTOCK_SERVER_URL`     | `http://localhost:8000`                                         | `apps/vnstock-server` base URL (VN equities, indices). |
| `COINGECKO_BASE_URL`     | `https://api.coingecko.com/api/v3`                              | CoinGecko API (crypto prices, USDT/VND).               |
| `COINGECKO_API_KEY`      | empty                                                           | CoinGecko demo key; raises the public rate limit.      |
| `CURRENCY_API_BASE_URL`  | `https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest` | currency-api, the forex fallback.                      |
| `YAHOO_FINANCE_BASE_URL` | `https://query1.finance.yahoo.com`                              | Yahoo Finance chart API (forex, world indices, VIX).   |

### 3.7 Minimal `.env.local`

Placeholders only:

```sh
GOOGLE_CLIENT_ID=<client-id>
GOOGLE_CLIENT_SECRET=<client-secret>
GOOGLE_SHEETS_ID=<spreadsheet-id>
# GOOGLE_REFRESH_TOKEN is written by `pnpm run auth:sheets`
```

Add the cache, AI and search variables as needed.

## 4. Nx targets

Run from the repo root with `pnpx nx run wealth-management:<target>`. pnpm only; never bun, npm or yarn.

| Target        | What it does                                                                                         |
| ------------- | ---------------------------------------------------------------------------------------------------- |
| `serve`       | Dev: runs `serve-api` and `serve-ui` in parallel.                                                    |
| `serve-api`   | Engine with live reload (`air`, `.air.toml`), port 8080 by default.                                  |
| `serve-ui`    | Vite dev server for the UI (default port 5173, `/api` proxied to `localhost:8080`).                  |
| `start`       | Prod-like: builds the UI (`build-ui:production`), then `go run .`; UI served by the engine at `/ui`. |
| `build`       | Go build of the engine.                                                                              |
| `build-ui`    | `vite build` into `ui/build/`.                                                                       |
| `test`        | Go unit and contract tests (`go test ./...`).                                                        |
| `test-ui`     | Vitest for the UI.                                                                                   |
| `check-ui`    | `svelte-check` type check.                                                                           |
| `lint`        | Go lint.                                                                                             |
| `lint-ui`     | ESLint for the UI.                                                                                   |
| `test-e2e`    | Live tests (`RUN_LIVE_API_TESTS=true`, `TestE2E*`). See section 6.                                   |
| `auth-sheets` | Google OAuth setup CLI (`cmd/google_oauth_setup`). Also `pnpm run auth:sheets`.                      |
| `verify-auth` | Checks the OAuth credentials and sheet access (`cmd/verify_auth`).                                   |

## 5. CLIs

- **`auth-sheets`** (`cmd/google_oauth_setup`): reads `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` (prompts if unset) and `GOOGLE_REDIRECT_URL` (default `http://127.0.0.1:3000`), prints a consent URL for the `spreadsheets` scope, asks for the redirected URL, and writes `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` and `GOOGLE_REFRESH_TOKEN` into the first existing `.env.local` (root first, else app-level; creates the root one if neither exists).
  With `--test` (`pnpx nx run wealth-management:auth-sheets -- --test`) it asks for the `spreadsheets` and `drive.file` scopes and writes `GOOGLE_TEST_REFRESH_TOKEN` instead, for the live e2e harness (WM-TEST-3). The production token keeps the `spreadsheets` scope only.
- **`verify-auth`** (`cmd/verify_auth`): reads `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REFRESH_TOKEN` and `GOOGLE_SHEETS_ID`, refreshes an access token, then opens the spreadsheet. `invalid_grant` means the refresh token expired or was revoked (a Google Cloud project in "Testing" mode expires tokens after 7 days): rerun `auth-sheets`.

## 6. Live E2E tests (disposable test sheet)

Live tests never touch the production sheet ([REQUIREMENTS.md §14.3](../REQUIREMENTS.md#143-disposable-sheet-copy-never-test-against-the-real-sheet)). Each run uploads a local `.xlsx` export of the workbook to Drive and converts it into a new Google Sheet. It starts the engine against that sheet and deletes it afterwards. The harness never opens the production sheet.

**One-time setup**:

1. Mint the test token:

   ```sh
   pnpx nx run wealth-management:auth-sheets -- --test
   ```

2. Export the workbook as `.xlsx` (File > Download > Microsoft Excel). Keep it **outside the repo**, because it holds real financial data.
3. Point the harness at it in `.env.local`:

   ```sh
   E2E_SOURCE_XLSX="~/projects/Money Manager.xlsx"
   ```

Re-export the workbook whenever the sheet's structure changes.

**Run**:

```sh
pnpx nx run wealth-management:test-e2e
# or, from apps/wealth-management:
RUN_LIVE_API_TESTS=true go test ./e2e/ -run '^TestE2E' -count=1 -v
```

Without `RUN_LIVE_API_TESTS=true` the suite is skipped, so `go test ./...` stays offline.

| Variable                    | Meaning                                                                                                     |
| --------------------------- | ----------------------------------------------------------------------------------------------------------- |
| `RUN_LIVE_API_TESTS`        | `true` opts in to the live suite.                                                                           |
| `E2E_SOURCE_XLSX`           | Path to the `.xlsx` export used as test data; a leading `~/` is expanded.                                   |
| `GOOGLE_TEST_REFRESH_TOKEN` | Test-only token from `auth-sheets --test`. Scopes: `spreadsheets` and `drive.file` (only files it created). |
| `KEEP_TEST_SHEET`           | `true` keeps the test sheet after the run for debugging; its name is logged.                                |

`GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` come from the same `.env.local` files as the engine. `GOOGLE_SHEETS_ID` is optional here. When it is set, it is used only to refuse the production sheet.

What a run does (`apps/wealth-management/e2e/harness_test.go`, `internal/e2esheet`):

1. **Janitor**: deletes leftover `wm-e2e-*` test sheets older than 24 h. Because of the `drive.file` scope, it can only see sheets that the harness itself created.
2. **Upload**: converts the workbook into a sheet named `wm-e2e-<runId>-<UTC timestamp>`. Tabs, formulas, `Prompts` and `Knowledge` come along. It aborts if the new id equals `GOOGLE_SHEETS_ID` or the name lacks the prefix.
3. **Engine**: builds and starts the engine with:
   - `WM_ENV_FILE=none` and `WM_TEST_MODE=true`;
   - the test sheet id and the test token;
   - `CACHE_KEY_PREFIX=wm-test:<runId>:`;
   - a random `API_TOKEN` and a free port on 127.0.0.1.

   In test mode the engine refuses to start unless the sheet title has the `wm-e2e-` prefix.

4. **Tests**, run serially on the one sheet:
   - Every sheet read route in the OpenAPI spec parses.
   - Structural checks.
   - Round-trips for accounts, transactions (including the stale ETag 409), idempotent batches, budget limits and Smart Capture rules.
   - Writes without the token get 401.
   - A market smoke test.

   Assertions are relative or structural, because the workbook holds real data that changes with each export.

5. **Teardown**: invalidates the run's cache keys and stops the engine. It deletes the test sheet unless `KEEP_TEST_SHEET=true`; before deleting, it re-checks the name prefix. The engine log is in `apps/wealth-management/tmp/e2e-<runId>/engine.log` and is kept when the run fails.
