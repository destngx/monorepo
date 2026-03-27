# VNStock Server as Primary Data Provider for Wealth Management

## Background

The wealth-management app currently fetches Vietnamese market data through a 3-adapter fallback chain: **CafeF → VNStock → Yahoo Finance**. CafeF only provides real-time index snapshots (no historical), VNStock calls the local `vnstock-server` Python API for individual stocks, and Yahoo Finance acts as the universal fallback (but is unreliable for VN tickers).

The `vnstock-server` FastAPI service already exposes `/api/v1/stocks/quote` and `/api/v1/stocks/historical` endpoints with built-in caching and rate limiting, backed by the `vnstock` Python library (source: VCI or KBS). It is the most reliable source for VN market data.

## Goal

Promote `vnstock-server` as the **primary data provider** for all Vietnamese market data (indices + individual stocks), consolidating the adapter chain:

**Before**: `CafeF (indices only) → VNStock (stocks only) → Yahoo Finance`
**After**: `VNStockServer (VN indices + stocks) → Yahoo Finance (fallback)`

## Tasks

### Task 1: Extend VNStock Server — Add Index & Price Board Endpoint Support

Add a new `/api/v1/stocks/index-history` endpoint to the vnstock-server for fetching index historical data (VN-Index, HNX, VN30, UPCOM) since the vnstock library supports index symbols natively.

**Files**:

- `apps/vnstock-server/src/modules/stocks/router.py`

### Task 2: Refactor VNStockAdapter — Support Indices + Health Check

Expand the existing `VNStockAdapter` to:

- Support VN index symbols (`^VNINDEX`, `^HNX`, `VN30`, `^UPCOM`) through the vnstock-server
- Map internal index symbols to vnstock-compatible format
- Add a health check to detect if vnstock-server is available before attempting requests
- Keep graceful fallback behavior (return `null` on failure so chain continues)

**Files**:

- `libs/wealth-management/src/services/data-sources/vnstock.ts`

### Task 3: Remove CafeFAdapter & Simplify Adapter Chain

Since VNStockAdapter will now handle all VN data (indices + stocks):

- Remove `CafeFAdapter` from the adapter chain in `market-data-service.ts`
- Update the `dataSourceAdapters` array to be `VNStockAdapter → YahooFinanceAdapter`
- Remove the hardcoded CafeF real-time fetch in `fetchMarketGroup()` (lines 424-477)
- Replace with a unified VNStock-first approach

**Files**:

- `libs/wealth-management/src/services/services/market-data-service.ts`
- `libs/wealth-management/src/services/data-sources/index.ts`

### Task 4: Update Tests

- Update `adapters.test.ts` to reflect new VNStockAdapter index support
- Update `fallback-chain.test.ts` if chain order changes
- Ensure all tests pass

**Files**:

- `libs/wealth-management/src/services/data-sources/__tests__/adapters.test.ts`
- `libs/wealth-management/src/services/data-sources/__tests__/fallback-chain.test.ts`

### Task 5: Verification & Build Check

- Run `bun nx test wealth-management-lib`
- Run `bun nx build wealth-management`
- Verify no type errors with `bun nx lint wealth-management-lib`
