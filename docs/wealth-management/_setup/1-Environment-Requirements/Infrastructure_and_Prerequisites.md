# Infrastructure and Prerequisites

Toolchain and external services for `apps/wealth-management`. Settings and bootstrap variables are listed in the [setup README](../README.md#3-settings-reference).

## 1. Toolchain

- **Go** at the version in `apps/wealth-management/go.mod`, for the engine and its CLIs.
- **Node + pnpm** (version pinned by `packageManager` in the root `package.json`), for Nx and the SvelteKit UI. pnpm only; never bun, npm or yarn. Install with `pnpm install` at the repo root.
- **Python** for `apps/market-data-server` (FastAPI), the service that serves all market data to the engine. Run it locally, or point `MARKET_DATA_SERVER_URL` at a running instance.

## 2. External services

| Service                                                  | Required                          | Used for                                                                                                                                            |
| -------------------------------------------------------- | --------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| Google Sheets API v4                                     | Yes, for all ledger data          | System of record. Google Cloud project with the Sheets API and an OAuth client.                                                                     |
| Upstash Redis (REST)                                     | No                                | Cache. Without it reads go straight to the sheet.                                                                                                   |
| OpenAI-compatible AI provider                            | No, for AI features only          | AI use cases (e.g. the repo's `ai-gateway`). See REQUIREMENTS.md §10.1.                                                                             |
| Tavily / Exa                                             | No                                | Paid web search, tried before the keyless DuckDuckGo fallback.                                                                                      |
| market-data-server                                       | Yes, for market data              | `apps/market-data-server`, reached via `MARKET_DATA_SERVER_URL`. Sources and their keys are set on the server.                                      |
| vnstock, fmarket, Yahoo Finance, CoinGecko, currency-api | No (called by market-data-server) | Market data sources, routed and cached by market-data-server; see [Market provider capabilities](../../_technical/Market_Provider_Capabilities.md). |

## 3. Security

- Secrets live only in the engine settings store (a `0600` file in the user config directory; see the [setup README §2](../README.md#2-settings-no-env-file)), the API token included. Never commit, print or log them.
- For long-lived refresh tokens, set the Google Cloud OAuth consent screen to "Production"; in "Testing" mode tokens expire after 7 days.
- Prefer a **Desktop** OAuth client: it accepts the loopback callback on any port (dev UI or engine). A Web client must list each exact redirect URI ([setup README §3.8](../README.md#38-first-run)).
- Keep `HOST` on a loopback address, or set an API token on the Settings page (on localhost) before binding to another interface (WM-SEC-1).
