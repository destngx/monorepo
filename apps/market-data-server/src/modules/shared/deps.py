from vnstock import Company, Finance, Listing, Quote, Trading, change_api_key
import inspect
import logging
import functools
from typing import Any, Dict, Optional
from fastapi import HTTPException
from starlette.concurrency import run_in_threadpool
from ...config import ServerConfig, VnstockConfig
from ...cache import CacheManager, CacheConfig, TTL
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


# Clients from vnstock.api (the `Vnstock().stock()` facade is deprecated in 4.x).
# `source` defaults to VNSTOCK_SOURCE; pass one explicitly for provider-only methods.
def _source(source: Optional[str]) -> str:
    return source or ServerConfig.DEFAULT_SOURCE


def quote_client(symbol: str, source: Optional[str] = None):
    return Quote(source=_source(source), symbol=symbol)


def trading_client(source: Optional[str] = None):
    return Trading(source=_source(source))


def listing_client(source: Optional[str] = None):
    return Listing(source=_source(source))


def finance_client(symbol: str, period: str, source: Optional[str] = None):
    return Finance(source=_source(source), symbol=symbol, period=period)


def company_client(symbol: str, source: Optional[str] = None):
    return Company(source=_source(source), symbol=symbol)


def get_cache_manager():
    return _CACHE_MANAGER


def get_rate_limiter():
    return _RATE_LIMITER


def is_authenticated():
    return _AUTHENTICATED


def _cache_params(func, args, kwargs) -> Dict[str, Any]:
    bound = inspect.signature(func).bind(*args, **kwargs)
    bound.apply_defaults()
    params = {}
    for k, v in bound.arguments.items():
        if v is None or k in ("self", "cls"):
            continue
        # Tickers are case-insensitive upstream; share one cache entry.
        if k in ("symbol", "symbols") and isinstance(v, str):
            v = v.replace(" ", "").upper()
        params[k] = v
    return params


class UpstreamQuotaExceeded(Exception):
    """vnstock (vnai) refused the call because its own quota is exhausted."""


def _guard_upstream(func, args, kwargs):
    try:
        return func(*args, **kwargs)
    except SystemExit as e:
        # vnai raises SystemExit on quota exhaustion. Left alone it escapes the
        # routes' `except Exception` and can take the server down.
        raise UpstreamQuotaExceeded(str(e).strip().splitlines()[0] if str(e).strip() else "") from None


class RateLimited(Exception):
    """Our own vnstock rate limit refused the call."""


async def run_vnstock(func, *args, cost: int = 1, **kwargs):
    """
    Run a blocking vnstock call: acquire `cost` rate-limit slots, run it in
    the threadpool, and turn vnai's SystemExit into UpstreamQuotaExceeded.
    """
    limiter = get_rate_limiter()
    if limiter:
        for _ in range(cost):
            if not await limiter.acquire():
                raise RateLimited("vnstock rate limit exceeded")
    try:
        # vnstock is synchronous; keep it off the event loop.
        return await run_in_threadpool(_guard_upstream, func, args, kwargs)
    except UpstreamQuotaExceeded as e:
        logger.warning("vnstock quota exhausted: %s", e)
        if limiter:
            await limiter.record_response(429)
        raise


async def _call_upstream(func, args, kwargs, cost: int = 1):
    try:
        return await run_vnstock(func, *args, cost=cost, **kwargs)
    except RateLimited:
        raise HTTPException(status_code=429, detail="Rate limit exceeded") from None
    except UpstreamQuotaExceeded:
        raise HTTPException(status_code=429, detail="Upstream rate limit exceeded") from None


def upstream_endpoint(endpoint_name: str, ttl: TTL = None, cost: int = 1):
    """
    Wrap a synchronous endpoint that calls vnstock.

    Order: cache (memory -> Redis) -> single-flight -> rate limit -> threadpool.
    Only cache misses consume upstream quota. `ttl` is seconds, None for
    "until 5am GMT+7", or a callable of the request params returning either.
    `endpoint_name` must be unique per route since it namespaces the cache key.
    `cost` is the number of upstream calls the endpoint makes.
    """

    def decorator(func):
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            cache_manager = get_cache_manager()
            if not cache_manager:
                return await _call_upstream(func, args, kwargs, cost)

            params = _cache_params(func, args, kwargs)
            resolved = ttl(params) if callable(ttl) else ttl
            data, was_cached = await cache_manager.get_or_compute(
                endpoint_name,
                params,
                lambda: _call_upstream(func, args, kwargs, cost),
                ttl=CacheConfig.get_ttl(resolved),
                should_cache=lambda r: isinstance(r, dict) and bool(r.get("success")),
            )
            if was_cached and isinstance(data, dict):
                data["cached"] = True
            return data

        return wrapper

    return decorator
