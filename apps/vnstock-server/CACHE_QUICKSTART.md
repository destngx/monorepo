# ⚡ Cache Quick Start

## What's New

vnstock-server now caches stock data using **Upstash Redis** for 3x faster response times.

## Start Using It

```bash
cd /Users/ez2/projects/personal/monorepo/apps/vnstock-server
source .venv/bin/activate
uvicorn src.main:app --reload
```

## Test Cache

### 1. Check Cache Status

```bash
curl http://localhost:8000/api/v1/health/cache
```

Response shows cache is enabled and TTL configuration.

### 2. Get Stock Quote (First Time)

```bash
curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
```

Response: `"cached": false` (API call made, data stored in cache)

### 3. Get Same Quote Again (Within 60s)

```bash
curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
```

Response: `"cached": true` (Data from cache, instant response)

## Response Format

```json
{
  "success": true,
  "data": {"symbol": "VCB", "price": 98.50, ...},
  "authenticated": true,
  "cached": false,    ← NEW: true = from cache, false = from API
  "source": "VCI"
}
```

## Cache TTL (Time To Live)

| Endpoint   | TTL        | Meaning                           |
| ---------- | ---------- | --------------------------------- |
| Quote      | 60 seconds | Prices cached for 1 minute        |
| Historical | 1 hour     | Historical data cached for 1 hour |
| Listing    | 1 hour     | Stock list cached for 1 hour      |

## Configuration

Redis credentials already set in `.env.local`:

```
UPSTASH_REDIS_REST_URL=...
UPSTASH_REDIS_REST_TOKEN=...
```

No additional setup needed!

## Adjust Cache Duration

Edit `src/cache.py`:

```python
class CacheConfig:
    QUOTE_TTL = 60           # Change this to adjust quote cache
    HISTORICAL_TTL = 3600    # Change this for historical data
```

## Disable Cache (if needed)

In `.env.local`:

```bash
CACHE_ENABLED=false
```

## Check Cache Works

```bash
# 1. First call (miss)
time curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
# ~500ms (API call)

# 2. Second call (hit)
time curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
# ~10ms (cache)

# 3. Third call (hit)
time curl "http://localhost:8000/api/v1/stocks/quote?symbol=VCB"
# ~10ms (cache)
```

## Performance

- **Without Cache**: Every request takes 500ms+ (API calls)
- **With Cache**: First request 500ms, subsequent <20ms
- **Result**: 3x+ faster for repeated requests!

## Monitor Cache

View logs while running:

```
✓ Cache hit: vnstock:quote:symbol=VCB:source=VCI
✗ Cache miss: vnstock:quote:symbol=VCB:source=VCI
✓ Cached: vnstock:quote:symbol=VCB:source=VCI (TTL: 60s)
```

## Troubleshooting

**Cache not working?**

```bash
curl http://localhost:8000/api/v1/health/cache
```

If `cache_enabled: false`, check:

- `.env.local` has `UPSTASH_REDIS_REST_URL`
- `.env.local` has `UPSTASH_REDIS_REST_TOKEN`
- Upstash account is active

**Want to clear cache?**

- Cache auto-expires after TTL
- Or restart the server (cache is in-memory backup)

## Learn More

See `CACHING.md` for detailed documentation.

---

**That's it!** Your cache is working automatically. 🎉
