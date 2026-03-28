# Cache Management API

This document describes the cache clearing endpoint for vnstock-server using Upstash Redis.

## Overview

The vnstock-server uses Upstash Redis for distributed caching of stock market data. The cache management API provides an endpoint to clear all cached data without restarting the server.

## Endpoint

### Clear All Cache

**Endpoint**: `POST /api/v1/health/cache/clear`

Clears ALL cached data from Upstash Redis.

**Request**:

```bash
curl -X POST "http://localhost:8000/api/v1/health/cache/clear"
```

**Response** (Success):

```json
{
  "success": true,
  "message": "All cache data cleared successfully",
  "timestamp": "2026-03-28T10:30:45.123456",
  "endpoint": "/api/v1/health/cache/clear"
}
```

**Status Codes**:

- `200 OK`: Cache cleared successfully
- `500 Internal Server Error`: Failed to clear cache
- `503 Service Unavailable`: Cache manager not initialized or cache disabled

**Use Cases**:

- Emergency cache clearing after data corruption
- Manual cache reset before critical analysis
- Cache refresh after bulk data updates
- Testing and development

## Other Cache Endpoints

### Get Cache Status

**Endpoint**: `GET /api/v1/health/cache`

Get current cache status and configuration.

```bash
curl "http://localhost:8000/api/v1/health/cache"
```

**Response**:

```json
{
  "cache_enabled": true,
  "backend": "Upstash Redis",
  "ttl_config": {
    "quote": null,
    "listing": null,
    "historical": null,
    "finance": null,
    "company": null
  }
}
```

## Configuration

### Environment Variables

```bash
# Cache Control
CACHE_ENABLED=true
UPSTASH_REDIS_REST_URL=https://your-upstash-instance.upstash.io
UPSTASH_REDIS_REST_TOKEN=your_secure_token_here
```

### Setting Up Upstash Redis

1. Go to [https://upstash.com](https://upstash.com)
2. Create a Redis database
3. Copy the REST URL and token
4. Add them to your `.env.local`:
   ```bash
   UPSTASH_REDIS_REST_URL=https://...
   UPSTASH_REDIS_REST_TOKEN=...
   ```

## How It Works

### Cache Key Structure

Cache keys follow this pattern:

```
vnstock:{endpoint}:{param1}=value1:param2=value2
```

Example:

```
vnstock:quote:symbol=VCB:interval=1D
```

### TTL Strategy

All cache entries expire at **5:00 AM GMT+7 (Vietnam time)** daily. This ensures:

- Fresh data for each trading day
- Automatic cache refresh without manual intervention
- Consistent behavior across the system

### Redis Operations

The cache manager supports:

- `GET`: Retrieve cached data
- `SET`: Store data with TTL
- `DEL`: Delete specific keys
- `FLUSHALL`: Clear all data (clears everything from the Redis instance)

## Usage Examples

### Python

```python
import httpx

async def clear_cache():
    async with httpx.AsyncClient() as client:
        response = await client.post(
            "http://localhost:8000/api/v1/health/cache/clear"
        )
        return response.json()

# Usage
result = await clear_cache()
print(f"Cache cleared: {result['success']}")
```

### JavaScript/Node.js

```javascript
const clearCache = async () => {
  const response = await fetch('http://localhost:8000/api/v1/health/cache/clear', { method: 'POST' });
  return response.json();
};

// Usage
const result = await clearCache();
console.log(`Cache cleared: ${result.success}`);
```

### Shell Script

```bash
#!/bin/bash
curl -X POST "http://localhost:8000/api/v1/health/cache/clear"
```

## Troubleshooting

### "Cache manager not initialized"

**Problem**: Error "Cache manager not initialized"

**Solution**:

1. Verify `UPSTASH_REDIS_REST_URL` is set in `.env.local`
2. Verify `UPSTASH_REDIS_REST_TOKEN` is set in `.env.local`
3. Check server startup logs for cache initialization messages
4. Test Redis connectivity: `curl -H "Authorization: Bearer $TOKEN" $UPSTASH_REDIS_REST_URL/ping`

### "Cache disabled"

**Problem**: Error "Cache is not enabled"

**Solution**:

1. Verify `CACHE_ENABLED=true` in `.env.local`
2. Verify Redis credentials are properly configured
3. Restart the server after updating environment variables

### Connection Error

**Problem**: "Cannot connect to http://localhost:8000"

**Solution**:

1. Make sure vnstock-server is running
2. Check the server port matches (default: 8000)
3. Verify firewall allows connections to the port

## Implementation Details

### UpstashRedisClient

The low-level Redis client handles HTTP requests to Upstash:

```python
client = UpstashRedisClient(url, token)
await client.flushall()  # Execute FLUSHALL command
```

### CacheManager

Higher-level cache management with TTL handling:

```python
cache = CacheManager()
await cache.flush_all()  # Clear all cache entries
```

## API Implementation

The cache clearing endpoint is implemented in `src/modules/health/router.py`:

```python
@router.post("/cache/clear")
async def clear_all_cache():
    """Clear all cached data from Upstash Redis."""
    cache_manager = get_cache_manager()
    # ... validation ...
    success = await cache_manager.flush_all()
    return {"success": success, "message": "..."}
```

## Related Documentation

- [CACHING.md](./CACHING.md) - Caching strategy and TTL configuration
- [API_KEY_SETUP.md](./API_KEY_SETUP.md) - API key setup guide
- [QUICKSTART.md](./QUICKSTART.md) - Server quick start guide
