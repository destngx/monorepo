# Progress Log

## 2026-03-27: VNStock Server as Primary Data Provider

### What was done

- Added `/api/v1/stocks/index-history` endpoint to vnstock-server for VN index data (VNINDEX, HNX, VN30, UPCOM)
- Refactored `VNStockAdapter` with index symbol mapping, health check caching, and dedicated endpoint routing
- Reordered adapter chain: `VNStock (primary) → Yahoo (fallback) → CafeF (last resort)`
- Removed hardcoded CafeF stitching logic and `fetchVNIndicesFromCafeF()` from `market-data-service.ts`
- Updated adapter and fallback chain tests for new chain priority
- Fixed pre-existing type error in `cache.ts:53`

### Verification

- Fallback chain tests: 5/5 pass
- Build: `bun nx build wealth-management` ✅
