# Cache Clearing API - Implementation Summary

## What Was Created

A simple **POST endpoint** to clear all Upstash Redis cache in vnstock-server. No authentication required (as requested).

## Endpoint

```
POST /api/v1/health/cache/clear
```

### Quick Usage

```bash
# Clear all cache
curl -X POST "http://localhost:8000/api/v1/health/cache/clear"

# Response
{
  "success": true,
  "message": "All cache data cleared successfully",
  "timestamp": "2026-03-28T10:30:45.123456",
  "endpoint": "/api/v1/health/cache/clear"
}
```

## Files Modified

### 1. `src/cache.py`

- Added `flushall()` method to `UpstashRedisClient` class
  - Executes Redis FLUSHALL command via REST API
  - Clears ALL data from the Redis instance
- Added `flush_all()` method to `CacheManager` class
  - Wrapper that provides safety checks
  - Returns success/failure status

### 2. `src/modules/health/router.py`

- Added new endpoint: `POST /api/v1/health/cache/clear`
- Validates cache is enabled
- Calls `cache_manager.flush_all()` and returns result
- Error handling for disabled or uninitialized cache

### 3. `.env.example`

- Updated with cache configuration variables
- Documented Upstash Redis setup

## How It Works

1. **Client** calls: `POST /api/v1/health/cache/clear`

2. **Server** receives request:
   - Gets cache manager instance
   - Validates cache is enabled
   - Calls `cache_manager.flush_all()`

3. **Cache Manager** calls:
   - `UpstashRedisClient.flushall()`

4. **Redis Client** sends:
   - HTTP POST to Upstash REST API
   - Command: `["FLUSHALL"]`

5. **Upstash** clears all data

6. **Response** returns:
   ```json
   {
     "success": true,
     "message": "All cache data cleared successfully",
     "timestamp": "...",
     "endpoint": "/api/v1/health/cache/clear"
   }
   ```

## Status Codes

| Code | Meaning                           |
| ---- | --------------------------------- |
| 200  | Cache cleared successfully        |
| 500  | Error during clearing             |
| 503  | Cache not initialized or disabled |

## Testing

### Test Script

Run the test script to verify the API:

```bash
cd apps/vnstock-server
python test_cache_api.py
```

This checks:

- Cache status
- Cache clearing
- Rate limit status

### Manual Test

```bash
# 1. Check cache is enabled
curl "http://localhost:8000/api/v1/health/cache"

# 2. Clear cache
curl -X POST "http://localhost:8000/api/v1/health/cache/clear"

# 3. Check if successful
echo $?  # Should print 0 for success
```

## Environment Setup

Add to `.env.local`:

```bash
# Cache (Upstash Redis)
CACHE_ENABLED=true
UPSTASH_REDIS_REST_URL=https://your-upstash-instance.upstash.io
UPSTASH_REDIS_REST_TOKEN=your_upstash_token_here
```

## Important Notes

1. **FLUSHALL clears everything**: This deletes ALL data in your Redis instance, not just vnstock cache
2. **No authentication**: Anyone who can reach the endpoint can clear cache
3. **One-way operation**: Clearing cache cannot be undone, but data will be recached on next requests
4. **Safe to use**: Cached data is always regenerated from live APIs on cache miss

## Use Cases

- 🔄 **Refresh stale data**: Clear cache before running critical analysis
- 🧹 **Cleanup**: Remove old cache entries accumulated over time
- 🐛 **Troubleshooting**: Test if issues are cache-related
- ✅ **Testing**: Fresh cache state for test runs

## Documentation

- [CACHE_MANAGEMENT.md](./CACHE_MANAGEMENT.md) - Full API documentation
- [CACHING.md](./CACHING.md) - Caching strategy details
- [QUICKSTART.md](./QUICKSTART.md) - Server quick start

## What's Next (Optional)

If you want to enhance this later:

1. **Add authentication**: Use API key or JWT token
2. **Add pattern clearing**: Clear only specific cache keys
3. **Add metrics**: Track how often cache is cleared
4. **Add audit logging**: Log who cleared cache and when
