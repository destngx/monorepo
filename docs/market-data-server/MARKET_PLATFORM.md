# Market Data Platform: Design

Status: accepted, phases 0 and 1 done (2026-10-14)

`apps/market-data-server` (formerly `apps/vnstock-server`) grows from a vnstock wrapper into the single market data server for the monorepo. It runs on the user's private local machine. It owns every market source, the fallback between them, caching, stale-data handling and the local price store. wealth-management (WM) keeps no market layer: no providers, no routing, no source adapters. It only calls this server.

## 1. Decisions

| ID      | Decision                                                                                                                                                                                                                                                                                                                           |
| ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MDS-D1  | The server owns **all** market sources, Vietnamese and global: vnstock, fmarket, Yahoo Finance, CoinGecko, currency-api, the Algolab and TCBS MCP servers, and later others.                                                                                                                                                       |
| MDS-D2  | WM replaces its five providers, the routing service and the fmarket service with one HTTP client adapter that implements its existing market ports. WM's own routes, MCP tools and services keep their shape.                                                                                                                      |
| MDS-D3  | The public API is organised by capability (`/api/v2/...`), never by source. Every response says which source served it.                                                                                                                                                                                                            |
| MDS-D4  | Behaviour WM depends on today (section 4) is carried over as-is. Known defects (section 5) are fixed during the move, not copied.                                                                                                                                                                                                  |
| MDS-D5  | SQLite is the durable store: last-good copies, the daily price store, ingestion state and MCP OAuth tokens. The response cache is the in-process LRU (L1) plus an L2 that is Upstash Redis when configured and a local SQLite file otherwise, by default wealth-management's cache file with keys under `CACHE_KEY_PREFIX` (done). |
| MDS-D6  | Technical indicators and screening run in the server, next to the price store, and are exposed over the API.                                                                                                                                                                                                                       |
| MDS-D7  | The app is named `market-data-server` (done). `/api/v1` stays until WM no longer calls it.                                                                                                                                                                                                                                         |
| MDS-D8  | MCP servers (Algolab, TCBS) are **sources of this server**: it is their MCP client and exposes their data through `/api/v2` like any other source. They are not wired into an AI assistant.                                                                                                                                        |
| MDS-D9  | No API authentication: the server only runs on a private local machine.                                                                                                                                                                                                                                                            |
| MDS-D10 | Free tiers only: every source runs on a free plan or free account (vnstock Community key, public fmarket, Yahoo Finance and currency-api, CoinGecko demo key, Algolab Lite, TCBS customer account). A source is only added if it has a usable free tier; paid plans are out of scope.                                              |

## 2. Architecture

```
            WM (Go)                       other consumers (AI tools, scripts)
              │  HTTP /api/v2                       │
              ▼                                     ▼
┌───────────────────────────────────────────────────────────────────┐
│ API layer (FastAPI, Pydantic models, error mapping, stale header) │
├───────────────────────────────────────────────────────────────────┤
│ Capability services: ticker, fx, series, candles, bank rates,     │
│ funds, gold, fundamentals, listing, indicators, screener          │
├───────────────────────────────────────────────────────────────────┤
│ Router: per-capability chains, skip vs failure, last-good, TTLs   │
├──────────────┬───────────────────────────┬────────────────────────┤
│ Cache (L1 LRU│ Store (SQLite: last-good, │ Scheduler (post-close  │
│ + single-    │ daily candles, fund NAV,  │ ingestion, backfill)   │
│ flight)      │ ingestion state)          │                        │
├──────────────┴───────────────────────────┴────────────────────────┤
│ Source adapters, each with its own rate limiter, timeout, health: │
│ vnstock │ fmarket │ yahoo │ coingecko │ currencyapi │ algolab │tcbs│
└───────────────────────────────────────────────────────────────────┘
```

### 2.1 Source adapters

- One package per source under `src/sources/<name>/`. An adapter declares which capabilities it implements (Python `Protocol`s mirroring WM's `TickerProvider`, `CandleProvider`, ...), and raises `Unsupported` for symbols, pairs or timeframes it does not cover.
- Adapters only translate: upstream request in, canonical model out. No caching, no fallback.
- Each adapter has its own rate limiter (vnstock: 60/min, 10k/day on the Community key), timeout (10 s default) and health probe.
- Blocking libraries (vnstock) run in the threadpool; HTTP sources use one shared `httpx.AsyncClient` per source.

### 2.2 Router

Ported from WM `service/market_provider` (`_route.go`):

- Try the capability's chain in order. An adapter without the capability, or raising `Unsupported`, is **skipped** (not a failure). One raising `NotFound` is a **miss** (it knows the item does not exist, for example fmarket's error 40065 for an unknown fund code). Any other error is a **failure**. In every case the chain moves on.
- First success returns `{value, provider, fallback}`, where `fallback` lists earlier failures and misses.
- Nothing answered: upstream error (502) listing every failure, then every skip, if any adapter failed; else not found (404) if any missed; else (all skipped) validation error ("no provider covers ..."). Only the upstream error serves the last-good copy.
- Chains are configuration, not code (defaults in section 3.3).

### 2.3 Cache and stale data

- L1 in-memory LRU with per-entry TTL and single-flight, L2 Upstash Redis or SQLite (built in phase 0).
- On success, also write a last-good copy to SQLite (7 days).
- On failure, serve the last-good copy (even with `skip_cache`), unless the error is a validation error. Mark the response stale: `meta.stale=true` plus the `X-Data-Stale` header, so WM can pass it straight through to its own middleware.
- `skip_cache=true` bypasses the read, never the write.

### 2.4 Store and ingestion

- SQLite in WAL mode, file path from `MARKET_DB_PATH`.
- `daily_candles(symbol_id, day, open, high, low, close, volume)` with `PRIMARY KEY(symbol_id, day) WITHOUT ROWID`, prices as integer VND (about 28 bytes per row; whole market about 110 MB).
- Same layout for fund NAV history. Intraday bars only for a configured watchlist.
- A scheduler job runs after the close (15:30 ICT on trading days): fetch the latest bar for every tracked symbol and append. Backfill runs once per new symbol, inside the vnstock quota.
- Candle reads hit the store first and only fetch the missing tail upstream.

### 2.5 MCP sources

Algolab and TCBS publish market data as remote MCP servers (Streamable HTTP). The server talks to them as an MCP client:

- One shared client module (`src/sources/mcp.py`): session setup (`initialize`, `Mcp-Session-Id`), `tools/call`, reconnect on expired sessions; auth is a header on the source's httpx client. It is a minimal JSON-RPC client over httpx because adding the `mcp` SDK was not approved (MDS-Q8); TCBS's OAuth flow may still need the SDK.
- Each MCP source is an ordinary adapter (`src/sources/algolab/`, `src/sources/tcbs/`): it maps capabilities to tool calls and tool output to canonical models. Tool output is parsed and validated; free text never reaches the API.
- Tool calls count against the source's quota, so they go through the same cache, single-flight and rate limiter as every other upstream call. Listing tools and connecting are free on Algolab.

**Algolab** (`https://mcp.algolab.vn`, server `algolab-market-data` 2.1.0):

- Auth: API key as `Authorization: Bearer alm_...` (`ALGOLAB_API_KEY` in `.env.local`). `/free` needs no key but shares one 120/min, 1,000/day pool with every anonymous user (405 used by 15:00 on 2026-10-14), so it can run out because of other users. A free Lite account can create keys (the tier MDS-D10 allows); Plus 1,000/day, Pro 5,000/day and Max 20,000/day are paid. As of 2026-10-14 there is no key, so the adapter (`src/sources/algolab/`) uses `/free` and sits last in the equity ticker, equity series and candle chains, where it is only called when every other source failed; setting `ALGOLAB_API_KEY` switches it to the account endpoint. Ruled out: `/free` as a primary source, and the algolab.vn website API (terms section 4 bans automated collection outside the APIs and keys Algolab provides).
- Tools: `get_market_overview`, `get_price_history`, `get_quotes` (50 symbols), `get_company_profile`, `get_financial_statements`, `get_research_reports`, `get_news` (with sentiment), `get_technical_signals`, `get_macro`, `get_intelligence`, `search`, `call_function` (advanced: `get_foreign_flows_detail`, `screen_stocks`, `get_sector_valuation`, `get_derivatives_snapshot`, `get_macro_series`, `get_quota`, ...).
- Output: one text content item holding JSON with `status`, `data` or `error {code, message, hint}`, and `meta.quota {used, limit}`. Errors set `isError`. No `outputSchema`, so the adapter validates with its own Pydantic models.
- Fills what the vnstock Community key lacks: foreign flows, macro, broker research, news sentiment, technical signals, a screener, derivatives.

**TCBS** (`https://mcp.tcbs.com.vn/mcp/tcinvest/`):

- Auth: OAuth 2.1 authorization code + PKCE (S256), refresh tokens, dynamic client registration, `client_secret_post`; no client-credentials grant. The server registers itself once, then:
  1. `GET /api/v2/providers/tcbs/login` redirects the browser to TCBS `/tcinvest/authorize` with `redirect_uri=http://localhost:<port>/api/v2/providers/tcbs/callback`.
  2. The callback exchanges the code and stores the client registration and tokens in SQLite (`oauth_clients`, `oauth_tokens`).
  3. Calls refresh the access token before expiry or on 401; if the refresh fails, the adapter reports unhealthy with "login required" and the router skips it.
- Tool list and output format are unknown until the first login (MDS-Q1).

## 3. API v2

### 3.1 Envelope

```json
{
  "data": {},
  "meta": {
    "provider": "vnstock",
    "fallback": ["yahoo: timeout"],
    "cached": false,
    "stale": false,
    "as_of": "2026-10-14T08:00:00Z"
  }
}
```

Errors use `{"error": {"code": "...", "message": "..."}}` and fixed statuses: 400 validation (including "no provider covers"), 404 unknown symbol or fund, 429 our own quota exhausted, 502 every provider failed with no last-good copy, 503 server misconfigured.

### 3.2 Endpoints

Everything WM uses today, by capability. All GET endpoints accept `skip_cache`.

| Endpoint                                                      | Replaces in WM                                | Notes                                                                                     |
| ------------------------------------------------------------- | --------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `GET /api/v2/ticker?symbol&type`                              | `GetTicker` (equity, ifc, gold, crypto)       | `Ticker` model unchanged: price or bid/ask, currency, timestamp.                          |
| `GET /api/v2/fx?from&to`                                      | `GetExchangeRate`                             | Rejects rates older than 4 days and moves down the chain. Any ISO pair, not just two.     |
| `GET /api/v2/series?symbol&type`                              | `GetPriceSeries` (equity, gold_usd, forex)    | Kept for compatibility; new callers use candles.                                          |
| `GET /api/v2/candles?symbol&timeframe&days` or `&from&to`     | `GetCandles`                                  | 1h, 4h, 1d, 1w. 4h and VN-index daily/weekly aggregated from 1h in ICT. Cleaned bars.     |
| `GET /api/v2/bank-rates`                                      | `GetBankInterestRate`                         | `tenor` empty, `updated_at` is the table date.                                            |
| `GET /api/v2/funds?asset_type&mmf&page&page_size`             | fmarket `getProductsFilterNav`                | fmarket's `data` payload as is (see below). `page_size` at most 100.                      |
| `GET /api/v2/funds/{code}`                                    | fmarket `getProductDetails`                   | Code URL-escaped.                                                                         |
| `GET /api/v2/funds/{fund}/nav`                                | fmarket `getNavHistory`                       | `fund` is the product id or the code. Full history; NAV store in phase 4.                 |
| `GET /api/v2/funds/issuers`                                   | fmarket `getIssuers`                          |                                                                                           |
| `GET /api/v2/gold/products`                                   | fmarket `getGoldProducts`                     |                                                                                           |
| `GET /api/v2/gold/history?from&to`                            | fmarket `getGoldPriceHistory`                 | `{rows, summary}`: daily rows and fmarket's `extra` block (ring gold, USD rate, changes). |
| `GET /api/v2/fx/history?from&to&pair`                         | fmarket `getUsdRateHistory`                   | USD_VND only. Dates as `yyyy-MM-dd` or `yyyyMMdd` in ICT; default the last 30 days.       |
| `GET /api/v2/providers` and `/api/v2/providers/{name}/health` | provider health routes                        | Capabilities per source; per-source quota usage later.                                    |
| `GET /api/v2/equities/{symbol}/{company,financials,...}`      | `/api/v1/company/*`, `/api/v1/finance/*`      | Deferred: no consumer yet; `/api/v1` stays.                                               |
| `GET /api/v2/listing?exchange&group&industry`                 | `/api/v1/listing/*`                           | Deferred, as above.                                                                       |
| `GET /api/v2/indicators?symbol&timeframe&set`                 | WM market analysis technicals (WM-INV-12..16) | Phase 5.                                                                                  |
| `GET /api/v2/screener?...`                                    | new                                           | Phase 5, store-backed.                                                                    |

The fund and gold endpoints return fmarket's `data` payload unchanged, because the WM UI parses fmarket's own fields. Typed fund models are a later step: they need the UI parsers (`ui/src/lib/investments/fmarket/`) to move with them.

Chains can be overridden with `MARKET_CHAINS`, a JSON object with any of `ticker`, `fx`, `series` (maps), `candles`, `bank_rates`, `funds` (lists); `fx` takes `"*"` for every other pair. Code: `src/market/` (models, router, service, API, last-good store) and `src/sources/` (one package per source).

### 3.3 Default chains and TTLs

Carried over from WM `domain/market_capability.go`:

| Capability                              | Chain                                                                                                  | TTL                                    |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------ | -------------------------------------- |
| Ticker                                  | equity: vnstock → fmarket → algolab; ifc: fmarket → vnstock; gold: fmarket; crypto: coingecko → yahoo  | 60 s                                   |
| FX                                      | USD_VND: fmarket → yahoo → currencyapi; USDT_VND: coingecko → currencyapi; others: yahoo → currencyapi | 600 s                                  |
| Series                                  | equity: vnstock → fmarket → algolab; gold_usd: fmarket; forex: yahoo → fmarket                         | 300 s                                  |
| Candles                                 | vnstock → yahoo → fmarket → algolab (fmarket: SJC only; XAUUSD is Yahoo's `GC=F` alone)                | 300 s (closed ranges: until 5am ICT)   |
| Bank rates                              | fmarket                                                                                                | 3600 s                                 |
| Funds, gold                             | fmarket                                                                                                | 60 s (WM's current fmarket action TTL) |
| Fundamentals                            | vnstock → algolab (TCBS once evaluated)                                                                | until 5am ICT                          |
| Foreign flows, macro, research, signals | algolab                                                                                                | until 5am ICT (intraday: 60 s)         |

Sources in one chain must be interchangeable: a fallback may change the provider but never the units or the instrument. VN stocks, ETFs and warrants are in thousands of VND on every source (Yahoo and Algolab publish VND and are scaled), indices in points (`tests/test_market_units.py`). Where sources genuinely quote different things for the same request, the answer says which: exchange rates carry a `basis` (`bank_sell`, `market`, `reference`, `crypto_market`). For the same reason fmarket no longer serves XAUUSD candles (spot, close-only) behind Yahoo's COMEX future.

## 4. Behaviour to carry over

These are pinned by WM tests and must be reproduced with server-side tests before WM switches:

- **Symbols**: trim, upper-case, strip leading `^`. `VNINDEX`, `VN30`, `USDVND`, `SJC`, `XAUUSD`, `VIX` are special; `IsVNListing` is false for the last four.
- **Candles**: drop bars with close ≤ 0 or no time; fill missing open/high/low from close; sort; last bar wins on duplicate times. Aggregation buckets in ICT: 4h at 00/04/.../20, 1d at midnight, 1w from Monday. Lookback defaults per timeframe (60, 180, 730, 1825 days), capped at 3650. An empty result counts as a provider failure.
- **vnstock adapter**: naive bar times are ICT. `1h→1H`, `1d→1D`, `1w→1W`. 4.x quirks are in section 4.1.
- **fmarket adapter**: browser-like headers (`Origin`, `Referer`, `F-Language: vi`, Firefox UA); history dates as `yyyyMMdd` in ICT (any other format silently returns a stale window); histories arrive newest first. SJC ticker is bid/ask in VND, XAU is a USD price. SJC candle close is `askSjc`.
- **yahoo adapter**: symbol map (`VIX→^VIX`, `USDVND→USDVND=X`, `XAUUSD→GC=F`, `VNINDEX→^VNINDEX.VN`, plain tickers get `.VN`, symbols with `.-=^` pass through; VN30, HNXINDEX, UPCOMINDEX, SJC unsupported). VN index daily/weekly built from 1h bars with the lookback clamped to 729 days.
- **coingecko adapter**: pinned ids (BTC, ETH, USDT→tether, USDC, SOL), otherwise `/search` exact symbol match with the best market-cap rank, memoised. Optional demo key header.
- **currencyapi adapter**: `/v1/currencies/{from}.min.json`, rate at `[from][to]`, timestamp is the date at 00:00Z.

### 4.1 vnstock 4.x on the Community key (phase 1 report)

Checked live on 2026-10-14 with vnstock 4.0.9 and vnai 2.6.3 (Community: 60 requests/min, financial statements limited to 8 periods; guests get 20/min).

| Dataset                                                                          | VCI | KBS | Notes                                                           |
| -------------------------------------------------------------------------------- | --- | --- | --------------------------------------------------------------- |
| Daily / intraday history, indices                                                | yes | yes | Plain OHLCV only; change is derived from the previous close.    |
| Intraday trades                                                                  | yes | yes |                                                                 |
| Price board                                                                      | yes | yes | VCI returns MultiIndex columns (89 fields), KBS 30 flat fields. |
| Financial statements, ratios                                                     | yes | yes | Periods `quarter` / `year` only.                                |
| Company overview, shareholders, officers, subsidiaries, affiliates, news, events | yes | yes |                                                                 |
| Ratio summary, trading stats                                                     | yes | no  |                                                                 |
| Ownership, capital history, insider trading                                      | no  | yes |                                                                 |
| ICB industries, government bonds                                                 | yes | no  | KBS raises `NotImplementedError`.                               |
| Listing, bonds, covered warrants, futures                                        | yes | yes |                                                                 |
| Foreign trading, macro, price depth, dividends                                   | no  | no  | Sponsor packages only (`vnstock_data`); use Algolab.            |

Other 4.x facts the server relies on:

- vnstock is no longer on PyPI. `vnstock` and `vnai` are pinned by URL and sha256 to the vnstocks.com index; `vnii` is only needed by sponsor packages and is not installed.
- `Vnstock().stock()` is deprecated; the server uses the `vnstock.api` classes.
- vnai raises `SystemExit` when its quota is exhausted; the server maps it to 429 and backs off.
- vnai can write a usage block into `AGENTS.md` and assistant config files, but only after `vnstock.enable_agent()`. The server never calls it.

## 5. Defects fixed during the move

- WM returns 502 for validation errors on exchange rate, price series and bank rates. v2 returns 400 consistently.
- Exchange rates only work for two pairs; v2 serves any ISO pair through yahoo → currencyapi.
- fmarket history actions default to the last 30 days in the host's local time; v2 uses ICT.
- `getProductDetails` puts the fund code into the URL unescaped; v2 escapes it.
- fmarket rejects a NAV history request without `navPeriod` ("invalid date format"), so WM's `getNavHistory` failed; v2 sends `navPeriod`.
- Adapters raise `Unsupported` for symbols, pairs or resolutions they do not cover, so the fallback indicator lists real failures only.
- Ticker, rate, series and bank-rate MCP tools have no cache bypass; v2 supports `skip_cache` everywhere, so WM can add it.
- Dead `FmarketHandler` methods in WM get deleted with the rest of the layer.

## 6. Changes in wealth-management

Done (phase 3):

- New adapter `adapter/market/marketdata/` implementing `port.MarketProviderService` and `port.FmarketService` over `/api/v2`. It maps `meta.stale` to WM's `MarkStale`, and server 400 / 404 answers to `ErrValidation` / `ErrNotFound` (anything else is an upstream error). The market, fmarket and provider-health handlers answer through `upstreamError`, so those reach clients as 400 / 404 instead of the blanket 502 they used to return.
- Deleted `adapter/market/{fmarket,vnstock,coingecko,currencyapi,yahoo}`, `service/market_provider/`, the routing config in `domain/market_capability.go`, candle cleaning and aggregation, and the per-capability provider ports. `httpjson` moved to `adapter/httpjson` for the web search providers.
- The per-provider `market:<name>` health checks became one `market-data-server` check. WM no longer caches market answers; the server does.
- Settings: the six market URL and key settings (including `VNSTOCK_SERVER_URL`) collapse to `MARKET_DATA_SERVER_URL`. Source keys such as `COINGECKO_API_KEY` move to the server's `.env.local`.
- Routes, MCP tools and the UI stay as they are, so `main_parity_test.go` keeps passing. Later, fmarket actions can move to typed WM routes that mirror `/api/v2/funds/*`.
- Market analysis (pulse, analysis, AI tools) keeps calling `GetCandles` until phase 5, then switches to `/api/v2/indicators`.
- `docs/wealth-management/REQUIREMENTS.md` §9.2, `_technical/Market_Provider_Capabilities.md`, the setup docs and the market API docs point at this document.

## 7. Phases

| Phase | Scope                                                                                                                                                                     | Done when                                                                                       |
| ----- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- |
| 0     | Done: cache key collisions, cache before quota, threadpool, single-flight, TTL policy, SQLite L2 fallback, rename.                                                        | Test suite green in a local venv.                                                               |
| 1     | Done: vnstock 4.0.9, `vnstock.api` clients, broken v1 routes fixed or removed, capability report (section 4.1).                                                           | `/api/v1` behaves as before on 4.x; capability report written.                                  |
| 2     | Done: adapter, router, envelope, SQLite last-good. vnstock adapter plus `/api/v2/ticker`, `/candles`, `/series`.                                                          | Section 4 candle and vnstock behaviour covered by server tests.                                 |
| 3     | Done: port fmarket, yahoo, coingecko, currencyapi. All section 3.2 endpoints except indicators and screener. Switch WM to the client adapter and delete its market layer. | WM unit tests, `main_parity_test.go` and the live `e2e/market_test.go` pass against the server. |
| 4     | Daily price store, NAV store, post-close ingestion, backfill.                                                                                                             | Candle requests for stored ranges make zero upstream calls.                                     |
| 5     | Indicators and screener on the store. WM market analysis uses them.                                                                                                       | WM-INV-12..16 served from the server.                                                           |
| 6     | MCP sources: shared MCP client, Algolab adapter (API key), then TCBS adapter with the OAuth login flow (section 2.5). Can start right after phase 2.                      | Foreign flows, macro and research served via `/api/v2`; TCBS tools inventoried.                 |

## 8. Open questions

| ID     | Question                                                                                                                                                                                                                                                                                                                                 |
| ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| MDS-Q1 | TCBS MCP: refresh-token lifetime, whether its terms allow use from a personal server rather than an AI assistant, the tool list and whether tools return structured data. Answered by the first login in phase 6. The user guide (`https://help.tcbs.com.vn/tcbs-mcp-server/`) is behind a Cloudflare check that blocks scripted access. |
| MDS-Q7 | Algolab terms (`algolab.vn/terms`, `/library/policies`): caching, storing results in the price store, and which plan the daily volume needs.                                                                                                                                                                                             |
| MDS-Q8 | Add the official Python `mcp` SDK as a dependency or keep the minimal Streamable HTTP client in `src/sources/mcp.py` (in use for Algolab since 2026-10-14). Revisit for TCBS's OAuth flow.                                                                                                                                               |

Resolved on 2026-10-14: Q2 (section 4.1); Q3 licensing (private local use is within the Community licence); Q4 auth (none, MDS-D9); Q5 Upstash (optional L2 with SQLite fallback, MDS-D5); Q6 name (`market-data-server`, MDS-D7).
