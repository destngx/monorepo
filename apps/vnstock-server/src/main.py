"""
Updated FastAPI wrapper for vnstock with API key authentication.
Leverages VNSTOCK_API_KEY for higher rate limits and better reliability.
Includes Upstash Redis caching for improved performance.
Includes rate limit management to prevent hitting Community tier limits (60 req/min, 10k req/day).
"""

from fastapi import FastAPI, HTTPException, Query
from vnstock import Vnstock, change_api_key
from typing import Optional, List, Dict, Any
import pandas as pd
from datetime import datetime, timedelta
import logging
import os

from .config import VnstockConfig, ServerConfig
from .cache import CacheManager, CacheConfig
from .rate_limiter import RateLimiter

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Vnstock API Wrapper",
    description="A FastAPI wrapper for the vnstock library with API key authentication for better rate limits and reliability.",
    version="0.4.0",
)

_API_KEY = None
_AUTHENTICATED = False
_CACHE_MANAGER = None
_RATE_LIMITER = None

try:
    _API_KEY = VnstockConfig.get_api_key()
    _AUTHENTICATED = change_api_key(_API_KEY)
    if _AUTHENTICATED:
        logger.info("✓ API key loaded and authenticated successfully")
    else:
        logger.warning("⚠ API key provided but authentication failed")
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


@app.on_event("startup")
async def startup_event():
    try:
        VnstockConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    if _CACHE_MANAGER:
        if _CACHE_MANAGER.enabled:
            logger.info("Cache layer is operational")
        else:
            logger.warning("Cache layer is disabled")

    if _RATE_LIMITER:
        status = _RATE_LIMITER.get_status()
        logger.info(
            f"✓ Rate limiter ready: {status.tier} ({status.requests_per_minute} req/min, {status.requests_per_day} req/day)"
        )


@app.get("/health")
def health_check():
    return {
        "status": "ok",
        "service": "vnstock-server",
        "timestamp": datetime.now().isoformat(),
        "authenticated": _AUTHENTICATED,
    }


@app.get("/")
def read_root():
    return {
        "message": "Welcome to Vnstock API Wrapper. Access /docs for API documentation.",
        "version": "0.2.0",
    }


@app.get("/api/v1/stocks/listing")
def get_listing():
    """
    Get a list of all companies listed on the Vietnamese stock market.
    Uses authenticated endpoint if API key is available.
    """
    try:
        client = get_vnstock_client()
        df = client.stock(
            symbol="VND", source=ServerConfig.DEFAULT_SOURCE
        ).listing.all_symbols()

        return {
            "success": True,
            "data": df.to_dict(orient="records"),
            "source": ServerConfig.DEFAULT_SOURCE,
            "authenticated": _AUTHENTICATED,
        }
    except Exception as e:
        logger.error(f"Error fetching stock listing: {str(e)}")
        raise HTTPException(
            status_code=500, detail=f"Error fetching stock listing: {str(e)}"
        )


@app.get("/api/v1/stocks/quote")
async def get_quote(symbol: str = Query(..., description="Ticker symbol")):
    """
    Get the latest price and metadata for a specific symbol.
    Uses authenticated endpoint if API key is available for better rate limits.
    Results are cached for 60 seconds.
    Respects Community tier rate limits (60 req/min, 10k req/day).
    """
    cache_params = {"symbol": symbol, "source": ServerConfig.DEFAULT_SOURCE}

    if _CACHE_MANAGER and _CACHE_MANAGER.enabled:
        cached_data = await _CACHE_MANAGER.get("quote", cache_params)
        if cached_data:
            cached_data["cached"] = True
            cached_data["rate_limited"] = False
            return cached_data

    if not await _RATE_LIMITER.acquire():
        raise HTTPException(
            status_code=429,
            detail="Rate limit exceeded. Daily quota exhausted or in backoff period. Please try again later.",
        )

    try:
        client = get_vnstock_client()
        df = client.stock(
            symbol=symbol, source=ServerConfig.DEFAULT_SOURCE
        ).quote.history(
            start=(datetime.now() - timedelta(days=7)).strftime("%Y-%m-%d"),
            end=datetime.now().strftime("%Y-%m-%d"),
        )

        if df.empty:
            raise HTTPException(
                status_code=404, detail=f"No data found for symbol {symbol}"
            )

        latest = df.iloc[-1].to_dict()

        if "time" in latest and isinstance(latest["time"], (pd.Timestamp, datetime)):
            latest["time"] = latest["time"].strftime("%Y-%m-%d %H:%M:%S")

        response = {
            "success": True,
            "data": {
                "symbol": symbol,
                "price": latest.get("close"),
                "change": latest.get("change", 0),
                "time": latest.get("time"),
                "source": ServerConfig.DEFAULT_SOURCE,
            },
            "authenticated": _AUTHENTICATED,
            "cached": False,
            "rate_limited": False,
        }

        if _CACHE_MANAGER and _CACHE_MANAGER.enabled:
            await _CACHE_MANAGER.set(
                "quote", cache_params, response, ttl=CacheConfig.QUOTE_TTL
            )

        return response
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching quote for {symbol}: {str(e)}")
        raise HTTPException(
            status_code=500, detail=f"Error fetching quote for {symbol}: {str(e)}"
        )


@app.get("/api/v1/stocks/historical")
async def get_historical_data(
    symbol: str = Query(..., description="Ticker symbol (e.g., GMD, VCB)"),
    start_date: Optional[str] = Query(
        None, description="Start date in YYYY-MM-DD format"
    ),
    end_date: Optional[str] = Query(None, description="End date in YYYY-MM-DD format"),
    resolution: str = Query(
        "1D", description="Data resolution (1M, 5M, 15M, 30M, 1H, 1D)"
    ),
):
    """
    Get historical stock data for a specific symbol.
    Uses authenticated endpoint if API key is available for better data quality.
    Results are cached for 1 hour.
    Respects Community tier rate limits (60 req/min, 10k req/day).
    """
    if not end_date:
        end_date = datetime.now().strftime("%Y-%m-%d")
    if not start_date:
        start_date = (datetime.now() - timedelta(days=30)).strftime("%Y-%m-%d")

    cache_params = {
        "symbol": symbol,
        "start_date": start_date,
        "end_date": end_date,
        "source": ServerConfig.DEFAULT_SOURCE,
    }

    if _CACHE_MANAGER and _CACHE_MANAGER.enabled:
        cached_data = await _CACHE_MANAGER.get("historical", cache_params)
        if cached_data:
            cached_data["cached"] = True
            cached_data["rate_limited"] = False
            return cached_data

    if not await _RATE_LIMITER.acquire():
        raise HTTPException(
            status_code=429,
            detail="Rate limit exceeded. Daily quota exhausted or in backoff period. Please try again later.",
        )

    try:
        client = get_vnstock_client()
        df = client.stock(
            symbol=symbol, source=ServerConfig.DEFAULT_SOURCE
        ).quote.history(start=start_date, end=end_date)

        if df.empty:
            response = {
                "success": True,
                "data": [],
                "source": ServerConfig.DEFAULT_SOURCE,
                "authenticated": _AUTHENTICATED,
                "cached": False,
                "rate_limited": False,
            }
            if _CACHE_MANAGER and _CACHE_MANAGER.enabled:
                await _CACHE_MANAGER.set(
                    "historical", cache_params, response, ttl=CacheConfig.HISTORICAL_TTL
                )
            return response

        if "time" in df.columns:
            if pd.api.types.is_datetime64_any_dtype(df["time"]):
                df["time"] = df["time"].dt.strftime("%Y-%m-%d %H:%M:%S")

        response = {
            "success": True,
            "data": df.to_dict(orient="records"),
            "source": ServerConfig.DEFAULT_SOURCE,
            "authenticated": _AUTHENTICATED,
            "cached": False,
            "rate_limited": False,
            "range": {
                "start": start_date,
                "end": end_date,
                "records": len(df),
            },
        }

        if _CACHE_MANAGER and _CACHE_MANAGER.enabled:
            await _CACHE_MANAGER.set(
                "historical", cache_params, response, ttl=CacheConfig.HISTORICAL_TTL
            )

        return response
    except Exception as e:
        logger.error(f"Error fetching historical data for {symbol}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Error fetching historical data for {symbol}: {str(e)}",
        )


@app.get("/api/v1/health/cache")
def get_cache_status():
    if not _CACHE_MANAGER:
        return {
            "cache_enabled": False,
            "reason": "Cache manager not initialized",
        }

    return {
        "cache_enabled": _CACHE_MANAGER.enabled,
        "backend": "Upstash Redis" if _CACHE_MANAGER.enabled else "None",
        "ttl_config": {
            "quote": CacheConfig.QUOTE_TTL,
            "listing": CacheConfig.LISTING_TTL,
            "historical": CacheConfig.HISTORICAL_TTL,
        },
    }


@app.get("/api/v1/health/ratelimit")
def get_rate_limit_status():
    if not _RATE_LIMITER:
        return {
            "error": "Rate limiter not initialized",
        }

    return _RATE_LIMITER.get_status_dict()


@app.get("/api/v1/health/quota")
def get_quota_status():
    if not _RATE_LIMITER:
        return {
            "error": "Rate limiter not initialized",
        }

    status = _RATE_LIMITER.get_status()

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
