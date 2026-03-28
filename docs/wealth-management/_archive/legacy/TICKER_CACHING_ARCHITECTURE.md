# Ticker Caching Architecture

## Overview

Ticker analysis details in wealth-management implement a **30-day caching strategy** across multiple layers to reduce API calls while maintaining data freshness. This document describes the complete architecture, TTL breakdown, cache invalidation flows, and monitoring.

## Caching Pyramid

```
┌─────────────────────────────────────────────────────────────┐
│         Layer 3: HTTP Response Cache                        │
│   (Browser/CDN Cache-Control headers)                       │
│   TTL: 30 days (2,592,000s)                                 │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ (miss)
┌─────────────────────────────────────────────────────────────┐
│         Layer 2: Service-Level Cache                        │
│   (In-memory getCached/setCache)                            │
│   - Stock Analysis: 30 days                                 │
│   - Asset Data: 14 days                                     │
│   - Search Results: 14 days                                 │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ (miss)
┌─────────────────────────────────────────────────────────────┐
│         Layer 1: Adapter-Level Cache                        │
│   (Data source adapters)                                    │
│   - VNStock Metadata: 30 days                               │
│   - VNStock Historical: 7 days                              │
│   - Yahoo Historical: 1 hour                                │
└─────────────────────────────────────────────────────────────┘
```

---

## Detailed TTL Breakdown

### Layer 1: Data Source Adapters

#### VNStock Adapter

- **Metadata** (company name, sector, market cap): **30 days**
  - Why: Company fundamentals rarely change between earnings reports
  - Risk: Very low — only changes on corporate actions (merge, split)
  - Invalidation: On stock split / sector change events

- **Historical Price Data** (OHLCV candles): **7 days**
  - Why: Technical patterns are stable across week, recomputation is cheap
  - Risk: Medium — trader might miss intraday moves, but analysis patterns hold
  - Invalidation: Automatic TTL expiry (no event needed)

#### Yahoo Finance Adapter

- **Historical Data**: **1 hour**
  - Why: Provides US stock data as fallback; 1h freshness for intraday analysis
  - Invalidation: TTL-based only

---

### Layer 2: Service-Level Cache

#### Stock Analysis Service (`stock-analysis-service.ts`)

- **TTL**: **30 days** with **±10% jitter** (27–33 days)
- **Cache key**: `stock-analysis:{symbol}`
- **What**: Complete analysis including technicals (RSI, EMA, trend), AI-generated summary, recommendation
- **Why 30 days**: Analysis synthesizes fundamental data (stable across quarter) + technical patterns (stable across week)
- **Jitter purpose**: Prevents cache stampede when 1000s of tickers expire simultaneously at same timestamp

```typescript
// Example TTL calculation with jitter
const ANALYSIS_CACHE_TTL = 30 * 24 * 3600; // 2,592,000 seconds
const jitterRange = Math.floor(ANALYSIS_CACHE_TTL * 0.1); // ±259,200s (±3 days)
const jitter = Math.floor(Math.random() * (jitterRange * 2)) - jitterRange;
const finalTTL = ANALYSIS_CACHE_TTL + jitter; // 27-33 days range
```

#### Asset Data Service (`market-data-service.ts`)

- **TTL**: **14 days**
- **Cache key**: `market-pulse:asset:{symbol}:{market}:{timeframe}`
- **What**: Processed market asset (price, momentum, % changes, OHLCV history)
- **Why 14 days**: 2 trading weeks allows pattern analysis to remain relevant

#### Search Results (`search-service.ts`)

- **TTL**: **14 days**
- **Cache key**: `search:{normalized_query}` (lowercase, spaces→underscores)
- **What**: Web search results from Tavily → Exa → DuckDuckGo
- **Why 14 days**: Search results remain relevant across 2 trading weeks; news + analysis links don't change dramatically
- **Normalization**: Query normalized to catch duplicate searches ("Apple Inc" vs "APPLE Inc")

---

### Layer 3: HTTP Response Cache

Both API endpoints include `Cache-Control` headers for browser/CDN caching:

```typescript
// /api/ticker/details - Company profile
{
  'Cache-Control': 'public, max-age=2592000, stale-while-revalidate=604800'
}
// 30 days cache, serve stale for 7 days if server unavailable

// /api/ticker/analyze - Technical analysis
{
  'Cache-Control': 'public, max-age=2592000, stale-while-revalidate=604800'
}
```

---

## Cache Invalidation Strategy

### 1. **TTL-Based (Passive Expiry)**

Default mechanism — caches expire automatically at TTL:

- **Stock Analysis**: Every 27–33 days (with jitter)
- **Asset Data**: Every 14 days
- **Search Results**: Every 14 days
- **Adapter Data**: 7 days (VNStock historical), 1 hour (Yahoo)

**No manual intervention required** — stale entries are cleaned up on next access.

### 2. **Event-Driven Invalidation (Active)**

For immediate invalidation when data becomes stale:

#### Earnings Release

```typescript
import { invalidateTickerCache } from '@wealth-management/utils';

// On earnings announcement for AAPL
await invalidateTickerCache('AAPL', 'earnings_release');
// Invalidates: stock-analysis:AAPL, vnstock:stock:AAPL, market-pulse:asset:AAPL:*
```

#### Stock Split

```typescript
await invalidateTickerCache('GOOG', 'stock_split');
// Invalidates all GOOG analysis (price-adjusted ratios need recalculation)
```

#### Sector Rotation

```typescript
import { invalidateCacheByTag } from '@wealth-management/utils';

// Invalidate all tech sector tickers
const techSymbols = ['AAPL', 'MSFT', 'NVDA', 'TSLA'];
for (const symbol of techSymbols) {
  await invalidateTickerCache(symbol, 'sector_rotation');
}
```

#### Tag-Based Batch Invalidation

```typescript
import { tagCache, invalidateCacheByTag } from '@wealth-management/utils';

// Tag all earnings-related analysis caches
await tagCache('earnings:Q4', 'stock-analysis:AAPL');
await tagCache('earnings:Q4', 'stock-analysis:MSFT');
await tagCache('earnings:Q4', 'market-pulse:asset:AAPL:1d');

// Later, invalidate all Q4 earnings analyses at once
await invalidateCacheByTag('earnings:Q4');
```

---

## Implementation Example: Complete Flow

```typescript
// 1. API REQUEST: POST /api/ticker/details?symbol=FPT

// 2. EXECUTION
app/api/ticker/details/route.ts:
  → const searchResponse = await executeSearch(query);
      // Hits search-service cache (key: "search:fpt_stock_profile...")
      // If miss: calls Tavily → caches for 14 days

  → return NextResponse.json(object, {
      headers: { 'Cache-Control': '...; max-age=2592000' }
    });
      // Browser caches response for 30 days

// 3. CACHE LAYERS ON MISS
   [Client Cache Miss]
     ↓
   [HTTP Layer Miss]
     ↓
   executeSearch()
     → getCached<SearchResponse>(cacheKey)
        // (1) Service-level cache hit? Return with [Cache hit]
        // (2) Miss? Proceed to (3)
        ↓
     → provider.execute(query)
        // Try Tavily API → Exa API → DuckDuckGo
        ↓
     → setCache(cacheKey, results, 14*24*3600)
        // Store in service cache for 14 days
        ↓
     → return results

// 4. ON EARNINGS RELEASE FOR FPT
   ChatAPI or Backend Event:
     → invalidateTickerCache('FPT', 'earnings_release');
        // Deletes: stock-analysis:FPT, vnstock:stock:FPT,
        //          vnstock:historical:FPT, market-pulse:asset:FPT:*
        // Next request regenerates analysis with fresh data

// 5. METRICS & MONITORING
   [Dashboard]
   Cache hit rate: 94% (target: >90%)
   Average response time (cache hit): 8ms
   Average response time (cache miss): 1200ms
   Invalidations today: 3 (earnings announcements)
   Total cached keys: 4,872
   Cache memory: 48MB (in-memory limit)
```

---

## Monitoring & Observability

### Key Metrics

**Cache Performance:**

- `cache_hit_rate`: % of requests served from cache (target: >90%)
- `cache_miss_rate`: % requiring fresh fetch
- `avg_response_time_hit`: Sub-10ms for cache hits
- `avg_response_time_miss`: 1–5s for API calls

**Cache Health:**

- `cache_size_bytes`: Total memory used (alert if >100MB)
- `cache_keys_count`: Total cached entries (alert if >10,000)
- `cache_expired_entries`: Entries cleaned up per day
- `invalidation_count_per_event_type`: Track event-driven invalidations

**Data Freshness:**

- `cache_ttl_violations`: Cases where data is >30 days old (should be 0)
- `invalidation_latency`: Time from event to cache cleared (target: <5s)

### Logging

Each cache operation includes logs:

```
[SearchService] ✓ Cache hit for query: "apple stock profile"
[SearchService] Cached results for 14 days (key: search:apple_stock_profile)
[VNStockAdapter] ✓ Cache hit for stock metadata: FPT
[MarketDataService] ✓ Cache hit for asset data: FPT (1d)
[Cache] Invalidated 7 keys for ticker FPT (reason: earnings_release)
```

### Alerts

Set up alerts for:

1. **Cache hit rate drops below 85%** — indicates cache thrashing or new tickers
2. **Cache invalidation rate spikes** — market event? data quality issue?
3. **Response time degradation** — cache misconfiguration or network issue
4. **Memory usage >100MB** — eviction policy needed

---

## API Reference

### Public Cache Functions

#### `getCached<T>(key: string): Promise<T | null>`

Retrieve value from cache if not expired.

```typescript
const cached = await getCached<StockAnalysis>('stock-analysis:FPT');
if (cached) return cached; // Serve cached
```

#### `setCache(key: string, data: any, ttlSeconds = 300): Promise<void>`

Store value in cache with TTL.

```typescript
await setCache('stock-analysis:FPT', analysis, 30 * 24 * 3600); // 30 days
```

#### `invalidateCache(keyPrefix: string): Promise<void>`

Delete all keys matching prefix.

```typescript
await invalidateCache('stock-analysis:'); // Clears all stock analysis
```

#### `tagCache(tag: string, key: string): Promise<void>`

Register a key with a tag for batch invalidation.

```typescript
await tagCache('earnings:FPT', 'stock-analysis:FPT');
```

#### `invalidateCacheByTag(tag: string): Promise<number>`

Delete all keys associated with a tag.

```typescript
const count = await invalidateCacheByTag('earnings:FPT');
console.log(`Invalidated ${count} keys`);
```

#### `invalidateTickerCache(symbol: string, reason: string): Promise<number>`

Delete all caches for a specific ticker.

```typescript
await invalidateTickerCache('FPT', 'earnings_release');
// Deletes: stock-analysis:FPT, vnstock:stock:FPT, market-pulse:asset:FPT:*
```

---

## Configuration

### Cache TTL Tuning

Adjust TTLs based on your use case:

```typescript
// Conservative: Shorter TTLs, fresher data, more API calls
const ANALYSIS_CACHE_TTL = 7 * 24 * 3600; // 7 days
const HISTORICAL_CACHE_TTL = 24 * 3600; // 1 day

// Aggressive: Longer TTLs, fewer API calls, risk of stale data
const ANALYSIS_CACHE_TTL = 60 * 24 * 3600; // 60 days
const HISTORICAL_CACHE_TTL = 30 * 24 * 3600; // 30 days
```

### Jitter Configuration

Fine-tune jitter to prevent cache stampede:

```typescript
const ANALYSIS_CACHE_TTL = 30 * 24 * 3600;
const jitterPercentage = 0.1; // ±10% (default)
const jitterPercentage = 0.05; // ±5% (less variance)
const jitterPercentage = 0.2; // ±20% (more spread)
```

---

## Testing

### Unit Tests for Cache Behavior

```typescript
// File: search-service.test.ts
describe('Search Service Caching', () => {
  it('should cache search results for 14 days', async () => {
    const results = await executeSearch('AAPL stock profile');
    const cached = await getCached(generateSearchCacheKey('AAPL stock profile'));
    expect(cached).toEqual(results);
  });

  it('should return cache hit on identical query', async () => {
    await executeSearch('AAPL');
    const secondCall = await executeSearch('AAPL');
    expect(secondCall.provider).toBeUndefined(); // No provider called
  });

  it('should invalidate cache on earnings', async () => {
    await invalidateTickerCache('AAPL', 'earnings');
    const cached = await getCached('stock-analysis:AAPL');
    expect(cached).toBeNull();
  });
});
```

### Integration Tests

```typescript
// File: ticker-api.integration.test.ts
describe('Ticker API Caching', () => {
  it('should serve ticker details from cache within 30 days', async () => {
    const res1 = await fetch('/api/ticker/details', { body: { symbol: 'FPT' } });
    expect(res1.headers.get('Cache-Control')).toContain('max-age=2592000');

    const res2 = await fetch('/api/ticker/details', { body: { symbol: 'FPT' } });
    expect(res2.status).toBe(200);
    expect(res2.headers.get('X-Cache')).toBe('HIT');
  });
});
```

---

## Troubleshooting

### Issue: High cache misses (hit rate < 85%)

**Possible causes:**

- Many unique queries/symbols with no repeats
- TTLs too short (data expiring too quickly)
- Cache size limit reached (evicting old entries)

**Solution:**

1. Check if symbol diversity is naturally high
2. Increase TTLs if data freshness allows
3. Monitor cache size and increase memory limit if needed

### Issue: Stale data returned (cache expired but not refreshed)

**Possible causes:**

- TTL calculation error
- Clock skew on server
- Manual invalidation not called when expected

**Solution:**

1. Verify cache TTL constants are set correctly
2. Check server NTP sync
3. Add event listeners for earnings announcements, stock splits

### Issue: Cache memory growing unbounded

**Possible causes:**

- Too many unique symbol/timeframe combinations
- TTLs too long
- Memory limit not enforced

**Solution:**

1. Implement LRU eviction policy
2. Reduce TTL or segment by date
3. Monitor cache size with `cache.size`

---

## Future Improvements

1. **Distributed Cache**: Move from in-memory to Redis for multi-instance deployments
2. **Cache Versioning**: Track data version, invalidate on schema changes
3. **Predictive Invalidation**: Pre-invalidate caches before earnings announce time
4. **Cache Warming**: Warm cache with popular tickers on startup
5. **Analytics**: Track cache efficiency per ticker (usage frequency vs TTL)

---

## References

- AWS Database Caching Strategies: https://docs.aws.amazon.com/whitepapers/latest/database-caching-strategies-using-redis/cache-validity.html
- Redis Best Practices: https://redis.io/guides/optimize-your-cache-for-fast-fresh-and-in-sync-data/
- Cache-Control HTTP Header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Cache-Control
- Exa API Caching: https://docs.exa.ai/reference/search-api-guide
- Tavily API: https://www.tavily.com/blog/tavily-101-ai-powered-search-for-developers
