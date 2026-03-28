# Cache TTL Policy: Dynamic Expiration at 5am GMT+7

## Overview

Both **vnstock-server** (Python) and **wealth-management** (TypeScript) now use **unified cache TTL policy**:

- **Max TTL**: 1 day (86400 seconds)
- **Reset time**: 5:00 AM GMT+7 (every day)
- **Backend**: Upstash Redis (distributed, serverless)
- **Fallback**: In-memory cache (wealth-management only)

## Architecture

### vnstock-server (Python)

**Files**:

- `src/ttl_utils.py` — TTL calculation utility
- `src/cache.py` — CacheManager with Upstash integration

**How it works**:

1. `calculate_ttl_until_next_5am()` computes seconds until next 5am GMT+7
2. CacheManager.set() uses dynamic TTL for all API responses
3. All cache keys stored in Upstash Redis expire at 5am daily

```python
from ttl_utils import calculate_ttl_until_next_5am

ttl = calculate_ttl_until_next_5am(max_ttl=86400)
await cache_manager.set(endpoint, params, data, ttl=ttl)
```

### wealth-management (TypeScript)

**Files**:

- `libs/wealth-management/src/utils/ttl-utils.ts` — TTL calculation
- `libs/wealth-management/src/utils/cache.ts` — Cache manager with Upstash + in-memory fallback

**How it works**:

1. `calculateTtlUntilNext5am()` computes seconds until next 5am GMT+7
2. UpstashCacheClient manages Upstash Redis connection
3. Fallback to in-memory cache if Upstash is unavailable
4. `getCached()`, `setCache()`, and invalidation functions handle both layers

```typescript
import { calculateTtlUntilNext5am } from './ttl-utils';

const ttl = calculateTtlUntilNext5am();
await setCache(key, data, ttl);
```

## Configuration

### Environment Variables

Both systems require Upstash credentials (already configured in `.env.local`):

```bash
UPSTASH_REDIS_REST_URL=https://...redis.upstash.io
UPSTASH_REDIS_REST_TOKEN=...token...
```

**Optional** (wealth-management):

- `CACHE_ENABLED` — Enable/disable caching (default: true)

## TTL Calculation Logic

The TTL is **dynamically calculated** until the next 5am GMT+7:

### Example 1: Current time is 3:00 AM GMT+7

```
Reset time today: 5:00 AM GMT+7
Time until reset: 2 hours = 7200 seconds
TTL returned: 7200 seconds
```

### Example 2: Current time is 6:00 AM GMT+7

```
Reset time today: 5:00 AM GMT+7 (already passed)
Reset time tomorrow: 5:00 AM GMT+7
Time until reset: 23 hours = 82800 seconds
TTL returned: 82800 seconds (capped at 86400)
```

### Example 3: Current time is 4:59 AM GMT+7

```
Reset time today: 5:00 AM GMT+7
Time until reset: 1 minute = 60 seconds
TTL returned: 60 seconds
```

## Cache Invalidation

### Full cache invalidation

```typescript
// wealth-management
await invalidateCache('accounts');  // Clear all keys starting with 'accounts'

// vnstock-server
await cache_manager.invalidate(endpoint='quote')  # Clear all 'quote' endpoint data
```

### Tag-based invalidation

```typescript
// wealth-management - useful for bulk invalidation
await invalidateCacheByTag('earnings:AAPL'); // Clear all AAPL analysis caches
```

### Ticker-specific invalidation

```typescript
// wealth-management - when earnings release or stock split occurs
await invalidateTickerCache('VCB', (reason = 'earnings-release'));
```

## Upstash Integration

### vnstock-server

Uses `httpx.AsyncClient` for async HTTP requests to Upstash REST API:

- GET: `/get/{key}`
- SET: `/pipeline` (with EX for TTL)
- DELETE: `/pipeline` (DEL command)

### wealth-management

Uses native `fetch()` API (Node.js 18+):

- GET: `/get/{key}`
- SET: `/pipeline` (with SET ... EX command)
- DELETE: `/pipeline` (with DEL command)

Both implementations fallback gracefully if Upstash is unavailable.

## Testing

### vnstock-server

```bash
python3 -c "
import sys
sys.path.insert(0, 'apps/vnstock-server/src')
from ttl_utils import calculate_ttl_until_next_5am
print(f'TTL: {calculate_ttl_until_next_5am()} seconds')
"
```

### wealth-management

```bash
node -e "
const ttl = 86400;  # Replace with actual import
console.log('TTL:', ttl, 'seconds');
"
```

## Migration Notes

### vnstock-server

- Removed per-type TTLs (quote=60, listing=3600, etc.)
- Now all data uses same dynamic TTL logic
- `CacheConfig.DEFAULT_TTL` = 86400 (hardcoded fallback)

### wealth-management

- Added Upstash Redis integration (optional)
- Preserved in-memory cache as fallback
- All `setCache()` calls now use dynamic TTL by default
- Existing code with explicit TTL still works: `setCache(key, data, 3600)`

## Monitoring

Both systems log cache operations:

```
[TTL] Current time: 2026-03-27T22:57:41+07:00 | Next reset: 2026-03-28T05:00:00+07:00 | TTL: 21738s (6.0h)
[Cache] Stored: vnstock:quote:symbol=VCB (TTL: 21738s, expires at 5am GMT+7)
[Upstash] GET error for key: connection timeout (fallback to in-memory)
```

Enable debug logging to see detailed cache operations:

- vnstock-server: Set `DEBUG=true` environment variable
- wealth-management: Check browser console (console.debug enabled in dev)

## Security Considerations

1. **No sensitive data in cache keys** — Cache key format includes parameters, avoid exposing secrets
2. **Upstash connection** — Uses HTTPS + Bearer token authentication
3. **TTL enforced** — Redis automatically deletes expired keys, no manual cleanup needed
4. **Fallback safety** — In-memory cache has max 1-day TTL, automatically expires on process restart

## FAQ

**Q: What happens if Upstash is down?**
A: wealth-management falls back to in-memory cache. vnstock-server logs a warning and continues without caching.

**Q: Can I override the TTL?**
A: Yes, pass explicit `ttl` parameter: `setCache(key, data, 3600)` (wealth-management) or `set(..., ttl=3600)` (vnstock-server). But default is always dynamic until 5am GMT+7.

**Q: Why 5am GMT+7?**
A: This is the stock market data refresh time in Vietnam (approximately market close + overnight processing).

**Q: How do I clear cache manually?**
A: Use invalidation functions: `invalidateCache()`, `invalidateCacheByTag()`, `invalidateTickerCache()`
