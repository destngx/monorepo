# Market Data Server API Walkthrough

The single market-data API of the monorepo. It owns every source (vnstock, fmarket, Yahoo Finance, CoinGecko, currency-api, Algolab), the fallback chains between them, caching and last-good copies. Consumers such as wealth-management call it and implement no market layer of their own. Design: [docs/market-data-server/MARKET_PLATFORM.md](../../docs/market-data-server/MARKET_PLATFORM.md).

## Base URL

- `http://localhost:8000/api/v2`: the multi-source market API (below).
- `http://localhost:8000/api/v1`: vnstock wrappers (listings, finance, company).

## Authentication

The server has no API auth; it runs on a private local machine. Put `VNSTOCK_API_KEY` in `.env.local` (loaded at startup) to get the Community quota of 60 requests/min instead of the guest 20/min.

## Setup

```sh
pnpx nx run market-data-server:install   # creates .venv, installs requirements.txt
pnpx nx run market-data-server:serve
pnpx nx run market-data-server:test
```

vnstock 4.x is not on PyPI. `requirements.txt` pins `vnstock` and `vnai` by URL and sha256 to the vnstocks.com index.

## Caching

Responses go through a two-tier cache: an in-process LRU (always on) in front of Upstash Redis when `UPSTASH_REDIS_REST_URL` / `UPSTASH_REDIS_REST_TOKEN` are set, otherwise a local SQLite file (`CACHE_SQLITE_PATH`). The default is wealth-management's cache file, `<system temp>/wealth-management/cache.db` (`$TMPDIR` on macOS, `/tmp` on Linux), and both apps use the same table, so one file holds both caches. L2 keys start with `CACHE_KEY_PREFIX` (default `mds:`; wealth-management uses `wm:`). `CACHE_ENABLED=false` keeps only the in-process tier. Concurrent requests for the same uncached key share one upstream call, and only cache misses count against the vnstock rate limit.

- Quotes, intraday, price board: 1 minute
- Listings: 1 hour
- Historical (range includes today): 5 minutes
- Historical (range ended before today): until 5am GMT+7
- Finance and company: until 5am GMT+7

Every entry also expires at the next 5am GMT+7 reset. `POST /api/v1/health/cache/clear` clears the in-process tier and this server's keys in L2 (those under `CACHE_KEY_PREFIX`); wealth-management's keys in a shared SQLite file or Upstash database stay.

## Market API (v2)

Every answer is an envelope that says where the data came from:

```json
{"data": {...}, "meta": {"provider": "fmarket", "fallback": ["vnstock: quota exceeded"], "cached": false, "stale": false, "as_of": "2026-10-14T08:00:00Z"}}
```

- `fallback` lists the sources that failed before `provider` answered.
- When every source fails, the last good answer (kept up to 7 days in `MARKET_DB_PATH`) is served with `stale: true` and an `X-Data-Stale` header.
- Errors are `{"error": {"code", "message"}}`: 400 `validation`, 404 `not_found` (unknown fund code or provider), 502 `upstream` when every source failed and there is no copy. A vnstock quota hit is a source failure, so the chain falls back.
- Every endpoint takes `skip_cache=true` to bypass the cache read.

| Endpoint                                       | Purpose                                                                                                             |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------- |
| `GET /ticker?symbol=&type=`                    | Current price. `type`: `equity`, `ifc` (fund code), `gold` (`SJC`, `XAU`), `crypto` (USD)                           |
| `GET /fx?from=&to=`                            | Exchange rate, for example `USD`/`VND` or `USDT`/`VND`, with its `basis` (below)                                    |
| `GET /fx/history?from=&to=`                    | fmarket's daily USD/VND rates (`pair=USD_VND` only)                                                                 |
| `GET /series?symbol=&type=`                    | Price series. `type`: `equity`, `gold_usd`, `forex`                                                                 |
| `GET /candles?symbol=&timeframe=&days=`        | OHLCV bars. `timeframe`: `1h`, `4h` (built from 1h), `1d`, `1w`; or `from`/`to` (YYYY-MM-DD, ICT) instead of `days` |
| `GET /bank-rates`                              | Deposit rates by bank and tenor                                                                                     |
| `GET /funds?asset_type=&mmf=&page=&page_size=` | fmarket fund list (raw fmarket payload, `page_size` at most 100)                                                    |
| `GET /funds/issuers`                           | Fund issuers                                                                                                        |
| `GET /funds/{code}`                            | Fund details                                                                                                        |
| `GET /funds/{fund}/nav`                        | Full NAV history; `fund` is a product id or a fund code                                                             |
| `GET /gold/products`                           | fmarket gold products                                                                                               |
| `GET /gold/history?from=&to=`                  | `{rows, summary}`: daily gold prices and today's ring gold, USD rate and changes                                    |
| `GET /providers`                               | Sources and their capabilities                                                                                      |
| `GET /providers/{name}/health`                 | One source's health                                                                                                 |

The fund and gold endpoints return fmarket's own `data` payload because consumers parse fmarket fields.

### Provider chains

Each capability tries its sources in order; a source that does not cover the request is skipped.

| Capability              | Default chain                                                                                           |
| ----------------------- | ------------------------------------------------------------------------------------------------------- |
| Ticker                  | equity: vnstock, fmarket, algolab; ifc: fmarket, vnstock; gold: fmarket; crypto: coingecko, yahoo       |
| FX                      | USD_VND: fmarket, yahoo, currencyapi; USDT_VND: coingecko, currencyapi; other pairs: yahoo, currencyapi |
| Series                  | equity: vnstock, fmarket, algolab; gold_usd: fmarket; forex: yahoo, fmarket                             |
| Candles                 | vnstock, yahoo, fmarket, algolab                                                                        |
| Bank rates, funds, gold | fmarket                                                                                                 |

`MARKET_CHAINS` (JSON) replaces any top-level entry, for example `MARKET_CHAINS='{"candles": ["yahoo", "vnstock"]}'`.

Cache lifetimes: ticker and funds 1 minute, series and candles 5 minutes, FX 10 minutes, bank rates 1 hour, each capped at the next 5am GMT+7 reset.

### Provider coverage and limits

All sources run on free tiers. Every source answers VN stocks, ETFs and warrants in thousands of VND (Yahoo and Algolab publish VND and are scaled) and indices in points, so a fallback never changes units (`tests/test_market_units.py`).

| Source                       | Serves in v2                                                                                                                                                                                                                            | Limits                                                                                                                                                                                                                                                                                                  |
| ---------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| vnstock (Community key)      | VN stock and index tickers (the latest daily bar of the last two weeks), equity series (30 days), 1h/1d/1w candles; v1 also has listings, financials and company data                                                                   | 60 requests/min and 10,000/day, shared with v1. No foreign flows, macro, order depth or dividends (Sponsor packages only). Financials capped at 8 periods. Ticker is the latest bar, not a live quote.                                                                                                  |
| fmarket                      | Fund (`ifc`) NAV tickers, SJC and XAU gold tickers, USD/VND bank sell rate and history, `gold_usd` and USD_VND `forex` series, SJC 1d/1w candles, bank rates, funds, NAV history, gold products and history                             | Unofficial public API with no published terms or limits. One quote per day, so SJC candles are close-only. The only source for funds, NAV, bank rates and SJC (SJC's own site is behind a Cloudflare challenge), so outages fall back to the stale copy. Skipped in the equity chains (no equity data). |
| Yahoo Finance                | FX market rates for ISO pairs, `forex` series, crypto tickers (`<COIN>-USD`, fallback), 1h/1d/1w candles for VN stocks, VNINDEX, VIX, USDVND, XAUUSD (COMEX front-month future `GC=F`, the only XAUUSD history) and Yahoo-style symbols | Unofficial API, may throttle or change without notice. No VN30, HNXINDEX, UPCOMINDEX or SJC. Hourly history only for the last 729 days; VN index 1d/1w bars are rebuilt from 1h. VN stock prices are scaled from VND to thousands.                                                                      |
| CoinGecko                    | Crypto tickers in USD, USDT/VND and coin/VND rates; 21 pinned coin ids, others resolved by search                                                                                                                                       | Free plan rate limits (higher with the optional `COINGECKO_API_KEY` demo key). An ambiguous symbol resolves to the best-ranked exact match.                                                                                                                                                             |
| currency-api                 | Last-resort FX for any ISO pair                                                                                                                                                                                                         | One daily reference snapshot, not a market rate. Rates older than 4 days are rejected.                                                                                                                                                                                                                  |
| Algolab (MCP, last fallback) | Live VN stock, ETF, warrant and index quotes, equity series (30 days), 1h/1d/1w candles                                                                                                                                                 | Without `ALGOLAB_API_KEY` it uses the keyless `/free` pool: 120 requests/min and 1,000/day shared with every anonymous user, so it can be exhausted by others. Each candle request costs 1-4 calls (1,000 bars per page). Personal, non-commercial use only.                                            |

Exchange rates carry a `basis`, because sources in one chain quote different rates for the same pair: `bank_sell` (fmarket, a bank's sell rate), `market` (Yahoo, the FX market), `reference` (currency-api, a daily snapshot) and `crypto_market` (CoinGecko). Copies cached before the field existed have no `basis`.

Global symbols use Yahoo's form (`BTC-USD`, `EURUSD=X`). Only symbols shaped like VN listings (3-10 letters and digits) reach vnstock and Algolab; a leading `^` is stripped, so Yahoo indices such as `^GSPC` are not reachable.

Gaps on every source today: fundamentals, foreign flows, macro, research and signals are not served by v2. The 4h timeframe is built from 1h bars by the service.

## vnstock API (v1)

### 1. Stock Listing

- `GET /listing/stocks`: All symbols.
- `GET /listing/bonds`: Corporate bonds.
- `GET /listing/covered-warrants`: Warrants.
- `GET /listing/future-indices`: Future indices.
- `GET /listing/government-bonds`: Gov bonds (VCI).
- `GET /listing/industries-icb`: ICB industry tree (VCI).
- `GET /listing/symbols-by-exchange?exchange=HOSE`: Filter by exchange.
- `GET /listing/symbols-by-group?group=VN30`: Filter by group.
- `GET /listing/symbols-by-industries?industry=8300`: Filter by ICB code or name (VCI).

### 2. Market Data

- `GET /stocks/quote?symbol=VND`: Current price and change.
- `GET /stocks/historical?symbol=VND&start_date=2024-01-01`: Historical OHLCV.
- `GET /stocks/intraday?symbol=VND`: Intraday price action.
- `GET /stocks/price-board?symbols=VND,SSI`: Real-time price board for multiple tickers.

### 3. Financial Statements

- `GET /finance/balance-sheet?symbol=VND&period=quarter`: Assets, liabilities, equity.
- `GET /finance/income-statement?symbol=VND&period=quarter`: Revenue, profit, expenses.
- `GET /finance/cash-flow?symbol=VND&period=quarter`: Operating, investing, financing flows.
- `GET /finance/ratio?symbol=VND&period=quarter`: Financial ratios.
- `GET /finance/get-all?symbol=VND&period=quarter`: Balance sheet, income statement and cash flow in one response (3 upstream calls).

`period` is `quarter` or `year`; `QUARTERLY` and `ANNUAL` are accepted. The Community key returns at most 8 periods.

### 4. Company Profile & Data

- `GET /company/overview?symbol=VND`: High-level overview.
- `GET /company/ratio-summary?symbol=VND`: Quick ratio view (VCI).
- `GET /company/trading-stats?symbol=VND`: Trading statistics (VCI).
- `GET /company/news?symbol=VND&page_size=20`: Recent news.
- `GET /company/events?symbol=VND`: Corporate events.
- `GET /company/officers?symbol=VND`: List of officers.
- `GET /company/shareholders?symbol=VND`: Main shareholders.
- `GET /company/ownership?symbol=VND`: Ownership by owner type (KBS).
- `GET /company/capital-history?symbol=VND`: Charter capital history (KBS).
- `GET /company/insider-deals?symbol=VND&page_size=20`: Insider trading (KBS).
- `GET /company/subsidiaries?symbol=VND`: List of subsidiaries.
- `GET /company/affiliate?symbol=VND`: Affiliated companies.

### 5. Health & Status

- `GET /health`: Basic health check.
- `GET /health/cache`: Cache configuration and backend status.
- `GET /health/ratelimit`: Current rate limit usage.
- `GET /health/quota`: Detailed quota analysis and next reset time.
