# Infrastructure and Prerequisites

Toolchain and external services for `apps/wealth-management`. Variables are listed in the [setup README](../README.md#3-environment-variables).

## 1. Toolchain

- **Go** at the version in `apps/wealth-management/go.mod`, for the engine and its CLIs.
- **Node + pnpm** (version pinned by `packageManager` in the root `package.json`), for Nx and the SvelteKit UI. pnpm only; never bun, npm or yarn. Install with `pnpm install` at the repo root.
- **Python**, only if you run `apps/vnstock-server` locally for Vietnamese equities.

## 2. External services

| Service                                         | Required                 | Used for                                                                                           |
| ----------------------------------------------- | ------------------------ | -------------------------------------------------------------------------------------------------- |
| Google Sheets API v4                            | Yes, for all ledger data | System of record. Google Cloud project with the Sheets API and an OAuth client.                    |
| Upstash Redis (REST)                            | No                       | Cache. Without it reads go straight to the sheet.                                                  |
| OpenAI-compatible AI provider                   | No, for AI features only | AI use cases (e.g. the repo's `ai-gateway`). See REQUIREMENTS.md §10.1.                            |
| Tavily / Exa                                    | No                       | Paid web search, tried before the keyless DuckDuckGo fallback.                                     |
| vnstock-server                                  | No, for VN equities      | `apps/vnstock-server`, reached via `VNSTOCK_SERVER_URL`.                                           |
| fmarket, CoinGecko, Yahoo Finance, currency-api | No (public APIs)         | Market data; see [Market provider capabilities](../../_technical/Market_Provider_Capabilities.md). |

## 3. Security

- Secrets live only in `.env.local` (gitignored) or the process environment. Never commit, print or log them.
- For long-lived refresh tokens, set the Google Cloud OAuth consent screen to "Production"; in "Testing" mode tokens expire after 7 days.
- Keep `HOST` on a loopback address, or set `API_TOKEN` before binding to another interface (WM-SEC-1).
