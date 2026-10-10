---
title: Wealth Management
summary: A Go and SvelteKit personal wealth platform that keeps the ledger in Google Sheets and adds caching, market data fallback routing, and an MCP interface on top.
year: 2026
status: active
stack: [Go, Fiber, SvelteKit, Google Sheets API, Upstash Redis, Python]
repo: https://github.com/destngx/monorepo/tree/main/apps/wealth-management-engine
featured: true
order: 3
sfx: ゴゴゴ
---

## The problem

I wanted to track accounts, transactions, budgets, and goals without giving up the spreadsheet. The ledger stays in Google Sheets, so it remains readable and editable outside the app. A backend sits in front of it to add caching, live market data, and a single API for the frontend and other tools.

## What it does

- REST route groups under `/api` for accounts, transactions, budgets, categories, goals, loans, tags, and investments.
- Market endpoints under `/api/market` for ticker quotes, exchange rates, price series, and bank rates. Passing `skipCache=true` bypasses the cache.
- `POST /api/sync` clears the cached data blocks so the next read reloads from Google Sheets.
- An MCP server runs in the same process over stdin/stdout and exposes the database and market services as tools.
- The SvelteKit dashboard is early. Its only screen calls `/api/health` to confirm the backend is reachable.

## How it's built

- The Go backend in `apps/wealth-management-engine` uses Fiber v2 and a hexagonal layout: `domain/`, `port/`, `service/`, and `adapter/`.
- The `adapter/db/google_sheets` package reads and writes rows through the Sheets API, and its tests cover token refresh.
- Redis caching goes through an Upstash REST client in `adapter/cache`.
- Market data comes from two providers, `vnstock` and `fmarket`, behind one `MarketProviderService`. The `vnstock` provider calls the Python `vnstock-server` over HTTP.
- The frontend is in `apps/wealth-management-dashboard`, built with SvelteKit and Tailwind classes.
- This replaces `apps/wealth-management-legacy`, the earlier Next.js version.

## Interesting bits

- Market data uses a fallback chain per data type. `DefaultMarketRoutingConfig` in `domain/market_capability.go` sets the order:

```go
GetTicker: map[TickerType][]string{
    TickerTypeEquity: {"vnstock", "fmarket"},
    TickerTypeIFC:    {"fmarket", "vnstock"},
    TickerTypeGold:   {"fmarket"},
},
```

- The first provider that succeeds wins, and successful results are cached for five minutes (`CacheTTLSeconds`).
- Bank rates only route to `fmarket`, so the chain has one entry.
- The backend treats Google Sheets as the source of truth and Redis as a disposable layer. Clearing the cache with `/api/sync` is the reset path.

## What's next

- Dashboard screens for accounts, transactions, and budgets. Only the health check exists today.
- Backend support for tax management and risk assessment. Both are specified in `docs/wealth-management/features`, but I found no routes or modules for them yet.
