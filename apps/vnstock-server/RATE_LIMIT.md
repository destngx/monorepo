# 🛡️ vnstock-server v0.4.0 - Rate Limit Management

**Status**: ✅ Production Ready | **Version**: 0.4.0 | **Date**: 2026-03-26

Comprehensive rate limit management for Community tier (60 req/min, 10k req/day) with automatic backoff, quota tracking, and circuit breaker protection.

## 🎯 What's New in v0.4.0

- ✅ **Rate Limit Queue** - Fair request distribution with exponential backoff
- ✅ **Quota Tracking** - Per-minute and per-day quota with persistent state
- ✅ **Circuit Breaker** - Fails gracefully when daily limit is exhausted
- ✅ **Monitoring Endpoints** - Real-time rate limit and quota status
- ✅ **Automatic Backoff** - Handles 429 (rate limited) responses from API
- ✅ **Graceful Degradation** - Works without rate limiter (permissive mode)

## 📊 Community Tier Limits

| Limit            | Value           | Notes                              |
| ---------------- | --------------- | ---------------------------------- |
| **Per Minute**   | 60 requests     | Hard limit, enforced by queue      |
| **Per Day**      | 10,000 requests | Soft limit with circuit breaker    |
| **Backoff**      | Exponential     | Auto-increases up to 5 minutes     |
| **Device Slots** | 1               | Only 1 device can use this API key |

### Tier Comparison

| Tier          | Per Min | Per Day | Cost | Status            |
| ------------- | ------- | ------- | ---- | ----------------- |
| **Community** | 60      | 10,000  | Free | Your current tier |
| Bronze        | 180     | 50,000  | Paid | Upgrade available |
| Silver        | 300     | 100,000 | Paid | Higher throughput |
| Golden        | 500     | 150,000 | Paid | Priority support  |
| Diamond       | 600     | 180,000 | Paid | Real-time data    |

## 🔧 How Rate Limiting Works

### Request Flow

```
Client Request
    ↓
Check Cache (Redis)
    ├─ HIT → Return cached + "cached": true + "rate_limited": false
    └─ MISS → Check Rate Limiter
              ├─ Over Quota? → HTTP 429 + Circuit Breaker
              └─ Available? → Wait for slot (exponential backoff)
                            ↓
                            Acquire Rate Limit Slot
                            ↓
                            Call vnstock API
                            ↓
                            Store in Redis Cache
                            ↓
                            Return data + "cached": false + "rate_limited": false
```

### Rate Limit Queue Mechanism

**Per-Minute Protection** (60 requests):

- Tracks timestamps of requests in current minute
- When limit is hit, waits for oldest request to leave window
- Sleep in 1-second chunks to remain responsive

**Per-Day Protection** (10,000 requests):

- Tracks all requests from last 24 hours
- Counts against daily quota
- WARNING: Once daily limit is hit, circuit breaker activates (no more API calls)

### Exponential Backoff Strategy

When API returns 429 (rate limited):

```
Backoff Duration: 5 × 2^n seconds (n increases with each 429)

Sequence:
- 1st 429 → 5 sec wait
- 2nd 429 → 10 sec wait
- 3rd 429 → 20 sec wait
- 4th 429 → 40 sec wait
- 5th 429 → 80 sec wait (caps at 300s / 5 min max)
```

**Reset Logic**: After successful request during backoff window, backoff multiplier resets to 1.0.

## 📍 Quota Tracking & Persistence

The rate limiter tracks quota across requests using environment state:

```python
# Persisted Quota Format
{
    "minute_requests": [1711442400.123, 1711442401.456, ...],  # Timestamps
    "day_requests": [1711356000.123, 1711356001.456, ...],      # Timestamps
    "timestamp": "2026-03-26T10:00:00"
}
```

**Automatic Cleanup**:

- Requests older than 1 minute are removed from per-minute tracking
- Requests older than 24 hours are removed from per-day tracking
- Cleanup runs before every quota check

## 🚨 Circuit Breaker Behavior

When daily quota (10,000) is exhausted:

```
Status: CIRCUIT BREAKER ACTIVATED
Action: All requests return HTTP 429
Error: "Rate limit exceeded. Daily quota exhausted. Try again after 24h"
Recovery: Automatic after 24 hours (when day window resets)
```

**Example**:

```python
# Daily quota hit at 3:30 PM
response = requests.get("http://localhost:8000/api/v1/stocks/quote?symbol=VCB")
# Status: 429
# Body: {"detail": "Rate limit exceeded. Daily quota exhausted..."}
# Will fail until 3:30 PM next day
```

## 📊 Monitoring Your Quota

### 1. Check Current Rate Limit Status

```bash
curl http://localhost:8000/api/v1/health/ratelimit
```

**Response**:

```json
{
  "tier": "COMMUNITY",
  "requests_per_minute": 60,
  "requests_per_day": 10000,
  "current_minute_requests": 5,
  "current_day_requests": 247,
  "remaining_minute": 55,
  "remaining_day": 9753,
  "minute_reset_at": "2026-03-26T10:02:30Z",
  "day_reset_at": "2026-03-27T10:02:30Z",
  "is_rate_limited": false,
  "backoff_until": null
}
```

### 2. Check Daily Quota with Warnings

```bash
curl http://localhost:8000/api/v1/health/quota
```

**Response**:

```json
{
  "tier": "COMMUNITY",
  "daily_quota": {
    "total": 10000,
    "used": 247,
    "remaining": 9753,
    "percentage_used": 2.5
  },
  "minute_quota": {
    "total": 60,
    "used": 5,
    "remaining": 55
  },
  "status": "ok",
  "warning_level": "normal",
  "resets_at": "2026-03-27T10:02:30Z"
}
```

**Warning Levels**:

- **normal**: > 80% remaining (good)
- **warning**: 10-20% remaining (caution)
- **critical**: < 10% remaining (urgent)

## 🛡️ Response Fields

All API responses now include rate limit info:

```json
{
  "success": true,
  "data": {...},
  "authenticated": true,
  "cached": false,
  "rate_limited": false    // NEW: Was rate limiter active?
}
```

- `"cached": true` - Data came from Redis cache (no API call)
- `"rate_limited": false` - Request did NOT hit rate limit
- `"rate_limited": true` - Would only appear in error response (HTTP 429)

## 🔄 Caching + Rate Limiting Strategy

**Smart Combination**:

| Endpoint   | Cache TTL | Strategy                              |
| ---------- | --------- | ------------------------------------- |
| Quote      | 60s       | Cache returns same-minute requests    |
| Historical | 1 hour    | Cache avoids repetitive large fetches |
| Listing    | 1 hour    | Cache avoids daily stock list refresh |

**Example Impact**:

```
100 users requesting VCB price simultaneously:
- Without cache: 100 API calls → 100 requests/min (over 60 limit)
- With cache (60s): 1 API call + 99 cache hits → 1 request/min ✓

Savings: 99% of requests served from cache!
```

## 📈 Usage Patterns to Avoid

### ❌ Pattern 1: Rapid Sequential Requests (Hammering)

```python
# BAD: This will hit rate limit quickly
for symbol in large_list:
    response = requests.get(f"api/v1/stocks/quote?symbol={symbol}")
    # 100 requests in 10 seconds → RATE LIMITED
```

**Solution**: Use cache or batch requests

```python
# GOOD: Space out requests or use cache
import asyncio
import httpx

async def fetch_quotes(symbols):
    async with httpx.AsyncClient() as client:
        tasks = [
            client.get(f"api/v1/stocks/quote?symbol={symbol}")
            for symbol in symbols
        ]
        # Respects rate limiter queue automatically
        results = await asyncio.gather(*tasks)
```

### ❌ Pattern 2: Daily Refresh at Same Time

```python
# BAD: Refreshing all 500 symbols at 9:30 AM
for symbol in ALL_SYMBOLS:  # 500 requests
    refresh_price(symbol)   # 500 requests in 10 min → daily quota consumed by 9:40 AM
```

**Solution**: Stagger refresh times

```python
# GOOD: Spread throughout trading hours
async def stagger_refresh():
    for i, symbol in enumerate(ALL_SYMBOLS):
        await asyncio.sleep(1)  # 1s delay between requests
        await fetch_quote(symbol)
```

## 🚀 Best Practices

### 1. **Always Check Quota Before Bulk Operations**

```python
import httpx

async def bulk_fetch(symbols):
    # Check quota first
    async with httpx.AsyncClient() as client:
        quota_resp = await client.get("http://localhost:8000/api/v1/health/quota")
        quota = quota_resp.json()

        if quota["daily_quota"]["remaining"] < len(symbols) * 2:
            print(f"⚠️ Warning: Only {quota['daily_quota']['remaining']} requests left")
            return None

        # Safe to proceed with bulk fetch
```

### 2. **Implement Retry Logic with Backoff**

```python
import httpx
import asyncio

async def fetch_with_retry(url, max_retries=3):
    for attempt in range(max_retries):
        try:
            async with httpx.AsyncClient() as client:
                resp = await client.get(url)
                if resp.status_code == 429:
                    wait = (2 ** attempt) * 5  # Exponential backoff
                    await asyncio.sleep(wait)
                    continue
                resp.raise_for_status()
                return resp.json()
        except Exception as e:
            if attempt == max_retries - 1:
                raise
            await asyncio.sleep(2 ** attempt)
```

### 3. **Cache-First Architecture**

```python
import httpx

async def get_quote_smart(symbol):
    # 1. Check cache first
    cache = redis.get(f"quote:{symbol}")
    if cache:
        return cache  # No API call needed!

    # 2. If cache miss, fetch from API
    async with httpx.AsyncClient() as client:
        resp = await client.get(f"api/v1/stocks/quote?symbol={symbol}")
        data = resp.json()

        # 3. Store in cache for 60s
        redis.setex(f"quote:{symbol}", 60, json.dumps(data))
        return data
```

## 🆙 Upgrading Your Tier

If Community tier limits are insufficient:

### Current Limits

```
60 requests/minute
10,000 requests/day
~$0/month (Free)
```

### Upgrade to Bronze

```
180 requests/minute (+200%)
50,000 requests/day (+400%)
~$29/month
```

### Contact vnstock Support

- Website: https://vnstocks.com (Vietnamese)
- Email: support@vnstock.com
- Support Hours: Business hours (Vietnam time)

## 🔍 Troubleshooting

### Issue: "Rate limit exceeded" immediately after startup

**Cause**: Persisted quota from previous session contains old requests
**Solution**: Clear quota or restart (quota auto-expires after 24h)

```bash
# Check if RATE_LIMIT_QUOTA is set
echo $RATE_LIMIT_QUOTA

# Clear it
unset RATE_LIMIT_QUOTA

# Restart server
uvicorn src.main:app --reload
```

### Issue: Hitting daily limit by noon

**Cause**: Too many concurrent users or inefficient caching
**Solution**:

1. Increase cache TTL (e.g., quote from 60s → 300s)
2. Implement request batching
3. Use read-only cache layer (no writes = fewer requests)
4. Upgrade to higher tier

### Issue: Backoff lasts 5+ minutes

**Cause**: Continuous rate limiting (API returning 429)
**Solution**:

1. Reduce concurrent requests
2. Implement longer delays between batches
3. Check if other users are hitting same limit
4. Contact vnstock support if API is overloaded

## 📝 Configuration Reference

### Rate Limiter Class

```python
from src.rate_limiter import RateLimiter

# Initialize with tier
limiter = RateLimiter(tier="COMMUNITY")

# Check if request can proceed
if await limiter.acquire():
    # Safe to make API call
    response = api_call()

    # Record response
    await limiter.record_response(response.status_code)

# Get status
status = limiter.get_status()
print(f"Remaining today: {status.remaining_day}")
print(f"Remaining this minute: {status.remaining_minute}")
```

### Rate Limiter Tiers

```python
class RateLimitTier(Enum):
    COMMUNITY = {"per_minute": 60, "per_day": 10000}
    BRONZE = {"per_minute": 180, "per_day": 50000}
    SILVER = {"per_minute": 300, "per_day": 100000}
    GOLDEN = {"per_minute": 500, "per_day": 150000}
    DIAMOND = {"per_minute": 600, "per_day": 180000}
```

## 🔗 Related Documentation

- [CACHING.md](./CACHING.md) - Cache strategy with rate limits
- [API_KEY_SETUP.md](./API_KEY_SETUP.md) - Authentication setup
- [CHANGES.md](./CHANGES.md) - Version history
- [QUICKSTART.md](./QUICKSTART.md) - Quick start guide

## 📊 Version History

| Version   | Date       | Changes                                    |
| --------- | ---------- | ------------------------------------------ |
| **0.4.0** | 2026-03-26 | Rate limit management with circuit breaker |
| 0.3.0     | 2026-03-26 | Caching with Upstash Redis                 |
| 0.2.0     | 2026-03-26 | API key authentication                     |
| 0.1.0     | -          | Initial release                            |

---

**Questions?** Check `/api/v1/health/ratelimit` for real-time status or review [API_KEY_SETUP.md](./API_KEY_SETUP.md) for support info.
