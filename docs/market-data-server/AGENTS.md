# Market Data Server: Engineering & Agent Standards

This document defines the core standards for `market-data-server` (Python), formerly `vnstock-server`.

## 🐍 Architecture: FastAPI & Market Data

The server is the single market-data API for the monorepo. It owns every source (vnstock, fmarket, Yahoo Finance, CoinGecko, currency-api, CafeF and Simplize bank rates, Zalopay and MoMo flexible savings, and the Algolab MCP as the last fallback; TCBS MCP is planned), the chains between them, caching and last-good copies; see [MARKET_PLATFORM.md](./MARKET_PLATFORM.md). Consumers such as wealth-management must not implement their own market layer: WM's only market adapter is `adapter/market/marketdata`, a client of `/api/v2`. It runs on a private local machine, so it has no API auth.

### 🗺️ Layout

- `src/modules/*`: `/api/v1`, thin vnstock wrappers (listings, finance, company, stocks, health).
- `src/market/`: `/api/v2`. `api.py` (routes, envelope, v2 error shape), `service.py` (input validation, cache, last-good fallback), `router.py` (chains), `chains.py` (default chains, `MARKET_CHAINS`, TTLs), `store.py` (last-good SQLite at `MARKET_DB_PATH`), `candles.py` (cleaning and aggregation in ICT), `models.py`, `symbols.py`, `errors.py`.
- `src/sources/<name>/provider.py`: one class per source. A source implements only the capabilities it has (`get_ticker`, `get_exchange_rate`, `get_price_series`, `get_candles`, `get_bank_rates`, `get_flexible_savings`, fund and gold methods) plus `health()` and `aclose()`. `src/sources/http.py` has the shared httpx client and `request_json`; `src/sources/nextjs.py` reads the `__NEXT_DATA__` page props of the sources that scrape Next.js pages (simplize, zalopay, momo), and `src/sources/rates.py` has `stated_rate` for a rate written in page prose; `src/sources/mcp.py` is the minimal MCP (Streamable HTTP) client used by MCP sources such as `algolab`. Algolab prices stocks in VND, so the adapter divides by 1000 to match the other sources; tests fake the MCP server in `tests/test_market_algolab.py`.

### 📁 Stack & Tooling

- **Language**: Python 3.10+ (developed on 3.14)
- **Framework**: FastAPI
- **Dependencies**: `requirements.txt` in a local `.venv` (`pnpx nx run market-data-server:install`).
- **vnstock**: 4.x. It is no longer published on PyPI; `vnstock` and `vnai` are pinned by URL and sha256 to the vnstocks.com index so pip never resolves those names elsewhere. Bump both together.
- **Caching**: in-process LRU, plus Upstash Redis when configured, otherwise a local SQLite file.

## 🔌 Calling vnstock

- Use the `vnstock.api` clients through the factories in `src/modules/shared/deps.py` (`quote_client`, `trading_client`, `listing_client`, `finance_client`, `company_client`). The `Vnstock().stock()` facade is deprecated in 4.x.
- Some methods exist on one provider only (VCI: `ratio_summary`, `trading_stats`, ICB data, government bonds; KBS: `ownership`, `capital_history`, `insider_trading`). Pass that source explicitly.
- Serialise results with `to_records` (`src/modules/shared/frames.py`); vnstock returns DataFrames, Series and MultiIndex columns.
- vnai raises `SystemExit` when its own quota is exhausted. `upstream_endpoint` turns that into a 429 and backs off; never catch `BaseException` in a route.
- In `/api/v2`, the vnstock source calls vnstock through `run_vnstock(func, *args, cost=...)` (`src/modules/shared/deps.py`), which takes rate-limit slots, runs the blocking call in the threadpool and turns vnai's `SystemExit` into an exception, so the chain falls back to the next source.

## 🔀 Market API (v2) Rules

- Sources raise `Unsupported` for a request they do not cover (the router skips them), `NotFound` when they know the item does not exist, and any other exception for a failure. Never return an empty or zero value instead of raising: the router only falls back on exceptions.
- When nothing answers, the router raises `UpstreamError` (502) if any source failed, else `NotFound` (404), else `ValidationError` (400). Only `UpstreamError` serves the last-good copy (`stale=true`, `X-Data-Stale`).
- Flexible savings is the one capability that is not a chain: `MarketService.flexible_savings` asks every source in `flexible_savings` on its own (own cache key and last-good copy) and lists the failures in `failed`.
- Validate input in `MarketService` before routing. Add a capability by implementing the method on its sources, a chain entry in `chains.py` and a route in `api.py` that returns `envelope(...)`.
- Fund and gold endpoints return fmarket's raw `data`, because consumers parse fmarket fields.
- A fallback must never change units or instrument. Sources return VN stocks, ETFs and warrants in thousands of VND and indices in points, scaling in the adapter if the upstream differs; add the source to `tests/test_market_units.py`. Exchange rates set `basis` (`RateBasis`).
- `is_vn_listing` only accepts VN-shaped symbols (3-10 letters and digits); VN-only sources raise `Unsupported` for anything else so they do not spend quota on global symbols.

## ⚡ Caching Policy

Endpoints that call an upstream use `upstream_endpoint(name, ttl, cost)` from `src/modules/shared/deps.py`, which applies, in order: in-memory LRU, the L2 tier, single-flight, rate limit (`cost` = upstream calls the endpoint makes), then runs the synchronous endpoint in a threadpool. Endpoint bodies must be plain `def` (vnstock is blocking), and `name` must be unique per route because it namespaces the cache key.

The L2 tier is Upstash Redis when `UPSTASH_REDIS_REST_URL` and `UPSTASH_REDIS_REST_TOKEN` are set, otherwise SQLite at `CACHE_SQLITE_PATH`, default wealth-management's cache file `<system temp>/wealth-management/cache.db`. The two apps share its `cache_entries` table, so its schema (expires_at in Unix milliseconds, NULL never expires) must stay identical to `apps/wealth-management/adapter/cache/sqlite_client.go`. Every L2 key carries `CACHE_KEY_PREFIX` (default `mds:`, never empty), and a cache clear deletes only keys under it: never `FLUSHALL` or an unscoped `DELETE`, because the database may hold wealth-management's keys. `CACHE_ENABLED=false` keeps only the in-process tier.

TTLs live in `CacheConfig` (`src/cache.py`) and are all capped at the next 5am GMT+7 reset:

- **Quotes / intraday / price board**: 1 minute
- **Listings**: 1 hour
- **Historical**: 5 minutes if the range includes today, otherwise until 5am GMT+7
- **Financial Statements / Company**: until 5am GMT+7

## 🧪 Testing & Workflow

- **Runner**: `pnpx nx run market-data-server:test` (or `PYTHONPATH=. .venv/bin/pytest` in the app directory).
- Unit tests patch the client classes in `src.modules.shared.deps` (for example `@patch("src.modules.shared.deps.Company")`). `tests/conftest.py` forces `CACHE_ENABLED=false` and `MARKET_DB_PATH=:memory:` so tests never touch a shared cache or store, and imports `src.main` so the shared services exist.
- v2 tests are async with anyio (`pytestmark = pytest.mark.anyio`). Sources are tested against `httpx.MockTransport` (the `Upstream` recorder in `tests/test_market_sources.py`), the router and service with `FakeSource`, and routes with `app.dependency_overrides[get_market_service]` and `TestClient`. No test calls a real upstream.
- **Environment**: `src/main.py` loads `.env.local` at startup (real environment variables win). Never commit or print secrets such as `VNSTOCK_API_KEY`.

## 🛠️ Implementation Rules

- Validate and normalise inputs at the route (tickers, resolutions, periods) and return 400 for bad values.
- Ensure error responses are descriptive and handle provider-specific exceptions gracefully.
