from fastapi import APIRouter, HTTPException
from datetime import datetime
from ...cache import CacheConfig
from ..shared.deps import get_cache_manager, get_rate_limiter, is_authenticated

router = APIRouter(prefix="/api/v1/health", tags=["Health"])


@router.get("")
def health_check():
    """General health check."""
    return {
        "status": "ok",
        "service": "vnstock-server",
        "timestamp": datetime.now().isoformat(),
        "authenticated": is_authenticated(),
    }


@router.get("/cache")
def get_cache_status():
    """Get cache status and configuration."""
    cache_manager = get_cache_manager()
    if not cache_manager:
        return {
            "cache_enabled": False,
            "reason": "Cache manager not initialized",
        }

    return {
        "cache_enabled": cache_manager.enabled,
        "backend": "Upstash Redis" if cache_manager.enabled else "None",
        "ttl_config": {
            "quote": CacheConfig.QUOTE_TTL,
            "listing": CacheConfig.LISTING_TTL,
            "historical": CacheConfig.HISTORICAL_TTL,
            "finance": CacheConfig.FINANCE_TTL,
            "company": CacheConfig.COMPANY_TTL,
        },
    }


@router.get("/ratelimit")
def get_rate_limit_status():
    """Get rate limiter status."""
    rate_limiter = get_rate_limiter()
    if not rate_limiter:
        return {"error": "Rate limiter not initialized"}
    return rate_limiter.get_status_dict()


@router.get("/quota")
def get_quota_status():
    """Get detailed quota status."""
    rate_limiter = get_rate_limiter()
    if not rate_limiter:
        return {"error": "Rate limiter not initialized"}

    status = rate_limiter.get_status()
    warning_level = "normal"
    if status.remaining_day < status.requests_per_day * 0.1:
        warning_level = "critical"
    elif status.remaining_day < status.requests_per_day * 0.2:
        warning_level = "warning"

    return {
        "tier": status.tier,
        "daily_quota": {
            "total": status.requests_per_day,
            "used": status.current_day_requests,
            "remaining": status.remaining_day,
            "percentage_used": round(
                100 * status.current_day_requests / status.requests_per_day, 1
            ),
        },
        "minute_quota": {
            "total": status.requests_per_minute,
            "used": status.current_minute_requests,
            "remaining": status.remaining_minute,
        },
        "status": "ok" if not status.is_rate_limited else "limited",
        "warning_level": warning_level,
        "resets_at": status.day_reset_at,
    }


@router.post("/cache/clear")
async def clear_all_cache():
    """Clear all cached data from Upstash Redis."""
    cache_manager = get_cache_manager()
    if not cache_manager:
        raise HTTPException(status_code=503, detail="Cache manager not initialized")

    if not cache_manager.enabled:
        raise HTTPException(status_code=503, detail="Cache is not enabled")

    try:
        success = await cache_manager.flush_all()
        return {
            "success": success,
            "message": "All cache data cleared successfully"
            if success
            else "Failed to clear cache",
            "timestamp": datetime.now().isoformat(),
            "endpoint": "/api/v1/health/cache/clear",
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Error clearing cache: {str(e)}")
