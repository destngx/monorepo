from vnstock import Vnstock, change_api_key
import logging
import functools
from fastapi import HTTPException
from ...config import VnstockConfig
from ...cache import CacheManager
from ...rate_limiter import RateLimiter

logger = logging.getLogger(__name__)

# Global instances (initialized in main.py startup or here if needed)
_API_KEY = None
_AUTHENTICATED = False
_CACHE_MANAGER = None
_RATE_LIMITER = None

def init_services():
    global _API_KEY, _AUTHENTICATED, _CACHE_MANAGER, _RATE_LIMITER
    try:
        _API_KEY = VnstockConfig.get_api_key()
        _AUTHENTICATED = change_api_key(_API_KEY)
        if _AUTHENTICATED:
            logger.info("✓ API key loaded and authenticated successfully")
    except ValueError as e:
        logger.warning(f"⚠ API key not configured: {e}")

    try:
        _CACHE_MANAGER = CacheManager()
    except Exception as e:
        logger.warning(f"⚠ Cache initialization failed: {e}")

    try:
        _RATE_LIMITER = RateLimiter(tier="COMMUNITY")
    except Exception as e:
        logger.error(f"⚠ Rate limiter initialization failed: {e}")
        raise

def get_vnstock_client():
    return Vnstock()

def get_cache_manager():
    return _CACHE_MANAGER

def get_rate_limiter():
    return _RATE_LIMITER

def is_authenticated():
    return _AUTHENTICATED

def rate_limit():
    """Decorator to apply rate limiting to an endpoint."""
    def decorator(func):
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            limiter = get_rate_limiter()
            if limiter:
                if not await limiter.acquire():
                    raise HTTPException(status_code=429, detail="Rate limit exceeded")
            return await func(*args, **kwargs)
        return wrapper
    return decorator

def cached_response(endpoint_name: str, ttl: int = 300):
    """Decorator to apply caching to an endpoint response."""
    def decorator(func):
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            cache_manager = get_cache_manager()
            if not cache_manager or not cache_manager.enabled:
                return await func(*args, **kwargs)

            # Build cache params from function arguments
            # This includes both positional and keyword arguments
            import inspect
            sig = inspect.signature(func)
            bound_args = sig.bind(*args, **kwargs)
            bound_args.apply_defaults()
            
            # Exclude 'self' or 'cls' if it's a method
            cache_params = {k: v for k, v in bound_args.arguments.items() if v is not None and k not in ('self', 'cls')}
            
            cached = await cache_manager.get(endpoint_name, cache_params)
            if cached:
                cached["cached"] = True
                return cached

            response = await func(*args, **kwargs)
            
            if isinstance(response, dict) and response.get("success"):
                await cache_manager.set(endpoint_name, cache_params, response, ttl=ttl)
            
            return response
        return wrapper
    return decorator
