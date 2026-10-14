from fastapi import APIRouter, HTTPException, Query
import logging
import pandas as pd
from datetime import datetime, timedelta
from typing import Optional
import re
from ...config import ServerConfig
from ...cache import CacheConfig
from ...ttl_utils import GMT_PLUS_7
from ..shared.deps import (
    is_authenticated,
    quote_client,
    trading_client,
    upstream_endpoint,
)
from ..shared.frames import to_records

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/stocks", tags=["Stocks"])


def validate_ticker_symbol(symbol: str) -> str:
    """
    Validate and normalize a ticker symbol.

    - Removes common prefixes like ^ (used in some APIs for indices)
    - Validates format (alphanumeric, hyphens, equals signs allowed)
    - Rejects invalid formats early

    Args:
        symbol: Raw ticker symbol from user input

    Returns:
        Normalized symbol (uppercase, cleaned)

    Raises:
        HTTPException: If symbol is invalid
    """
    if not symbol or not isinstance(symbol, str):
        raise HTTPException(status_code=400, detail="Symbol must be a non-empty string")

    # Clean the symbol: strip whitespace and remove common prefixes
    cleaned = symbol.strip().upper()

    # Remove common prefixes used in external APIs (^UPCOM becomes UPCOM)
    if cleaned.startswith("^"):
        cleaned = cleaned[1:]

    # Validate format: alphanumeric, hyphens, equals signs allowed
    # Vietnamese tickers can have special chars like = (VND=X) but those are invalid for vnstock
    if not re.match(r"^[A-Z0-9\-]+$", cleaned):
        raise HTTPException(
            status_code=400,
            detail=f"Invalid ticker symbol '{symbol}'. Must contain only letters, numbers, and hyphens.",
        )

    # Reject known invalid tickers
    if cleaned in ("IFC", "VND=X", "HNX", "UPCOM"):
        raise HTTPException(
            status_code=400,
            detail=f"'{symbol}' is not a valid vnstock ticker. Use stock symbols (e.g., VCB, SAB, GMD).",
        )

    # Length validation: most Vietnamese stocks are 3-4 chars
    if len(cleaned) < 2 or len(cleaned) > 10:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid ticker symbol '{symbol}'. Length must be 2-10 characters.",
        )

    return cleaned


# vnstock intervals are case-sensitive ("1M" is a month, "1m" a minute).
RESOLUTIONS = {"1m", "5m", "15m", "30m", "1H", "1D", "1W", "1M"}
RESOLUTION_ALIASES = {"1h": "1H", "1d": "1D", "1w": "1W"}


def validate_resolution(resolution: str) -> str:
    """Normalise a resolution to a vnstock interval, rejecting unknown values."""
    value = (resolution or "1D").strip()
    value = RESOLUTION_ALIASES.get(value, value)
    if value not in RESOLUTIONS:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid resolution '{resolution}'. Valid: {', '.join(sorted(RESOLUTIONS))}",
        )
    return value


def historical_ttl(params: dict) -> Optional[int]:
    """Ranges ending before today are final; ranges including today still move."""
    end_date = params.get("end_date")
    today = datetime.now(GMT_PLUS_7).strftime("%Y-%m-%d")
    if end_date and end_date < today:
        return CacheConfig.HISTORICAL_CLOSED_TTL
    return CacheConfig.HISTORICAL_LIVE_TTL


@router.get("/quote")
@upstream_endpoint("stocks-quote", ttl=CacheConfig.QUOTE_TTL)
def get_quote(symbol: str = Query(..., description="Ticker symbol")):
    """Get the latest price and metadata for a specific symbol."""
    try:
        symbol = validate_ticker_symbol(symbol)
        df = quote_client(symbol).history(
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
        # vnstock 4.x returns plain OHLCV, so derive the change from the prior close.
        change = 0.0
        if len(df) > 1 and pd.notnull(latest.get("close")):
            change = round(float(latest["close"]) - float(df["close"].iloc[-2]), 4)

        return {
            "success": True,
            "data": {
                "symbol": symbol,
                "price": latest.get("close"),
                "change": change,
                "time": latest.get("time"),
                "source": ServerConfig.DEFAULT_SOURCE,
            },
            "authenticated": is_authenticated(),
            "cached": False,
            "rate_limited": False,
        }
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching quote for {symbol}: {str(e)}")
        raise HTTPException(
            status_code=500, detail=f"Error fetching quote for {symbol}: {str(e)}"
        )


@router.get("/intraday")
@upstream_endpoint("stocks-intraday", ttl=CacheConfig.INTRADAY_TTL)
def get_intraday_data(
    symbol: str = Query(..., description="Ticker symbol"),
    page_size: int = Query(100, description="Number of records per page"),
):
    """Get intraday trading data."""
    try:
        symbol = validate_ticker_symbol(symbol)
        df = quote_client(symbol).intraday(page_size=page_size)
        return {"success": True, "data": to_records(df), "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/price-board")
@upstream_endpoint("stocks-price-board", ttl=CacheConfig.PRICE_BOARD_TTL)
def get_price_board(
    symbols: str = Query(..., description="Comma-separated ticker symbols"),
):
    """Get price board for multiple symbols."""
    try:
        symbols_list = [validate_ticker_symbol(s.strip()) for s in symbols.split(",")]
        df = trading_client().price_board(symbols_list=symbols_list)
        return {"success": True, "data": to_records(df), "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/historical")
@upstream_endpoint("stocks-historical", ttl=historical_ttl)
def get_historical_data(
    symbol: str = Query(..., description="Ticker symbol (e.g., GMD, VCB)"),
    start_date: Optional[str] = Query(
        None, description="Start date in YYYY-MM-DD format"
    ),
    end_date: Optional[str] = Query(None, description="End date in YYYY-MM-DD format"),
    resolution: str = Query(
        "1D", description="Data resolution (1m, 5m, 15m, 30m, 1H, 1D, 1W, 1M)"
    ),
):
    """Get historical stock data for a specific symbol."""
    interval = validate_resolution(resolution)
    if not end_date:
        end_date = datetime.now().strftime("%Y-%m-%d")
    if not start_date:
        start_date = (datetime.now() - timedelta(days=30)).strftime("%Y-%m-%d")

    try:
        symbol = validate_ticker_symbol(symbol)
        df = quote_client(symbol).history(
            start=start_date, end=end_date, interval=interval
        )

        if df.empty:
            return {
                "success": True,
                "data": [],
                "source": ServerConfig.DEFAULT_SOURCE,
                "authenticated": is_authenticated(),
                "cached": False,
                "rate_limited": False,
            }

        if "time" in df.columns:
            if pd.api.types.is_datetime64_any_dtype(df["time"]):
                df["time"] = df["time"].dt.strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "data": df.to_dict(orient="records"),
            "source": ServerConfig.DEFAULT_SOURCE,
            "authenticated": is_authenticated(),
            "cached": False,
            "rate_limited": False,
            "range": {
                "start": start_date,
                "end": end_date,
                "records": len(df),
            },
            "resolution": interval,
        }
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching historical data for {symbol}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Error fetching historical data for {symbol}: {str(e)}",
        )


@router.get("/index-history")
@upstream_endpoint("stocks-index-history", ttl=historical_ttl)
def get_index_history(
    symbol: str = Query(
        ..., description="Index symbol (e.g., VNINDEX, HNX, VN30, UPCOM)"
    ),
    start_date: Optional[str] = Query(
        None, description="Start date in YYYY-MM-DD format"
    ),
    end_date: Optional[str] = Query(None, description="End date in YYYY-MM-DD format"),
):
    """Get historical data for a Vietnamese market index."""
    VALID_INDICES = {"VNINDEX", "VN30"}

    normalized = symbol.upper().replace("^", "").replace("-", "")
    if normalized not in VALID_INDICES:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid index symbol '{symbol}'. Valid indices: {', '.join(sorted(VALID_INDICES))}",
        )

    if not end_date:
        end_date = datetime.now().strftime("%Y-%m-%d")
    if not start_date:
        start_date = (datetime.now() - timedelta(days=90)).strftime("%Y-%m-%d")

    try:
        df = quote_client(normalized).history(
            start=start_date,
            end=end_date,
        )

        if df.empty:
            return {
                "success": True,
                "data": [],
                "source": ServerConfig.DEFAULT_SOURCE,
                "authenticated": is_authenticated(),
                "cached": False,
            }

        if "time" in df.columns:
            if pd.api.types.is_datetime64_any_dtype(df["time"]):
                df["time"] = df["time"].dt.strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "data": df.to_dict(orient="records"),
            "source": ServerConfig.DEFAULT_SOURCE,
            "authenticated": is_authenticated(),
            "cached": False,
            "index": normalized,
            "range": {
                "start": start_date,
                "end": end_date,
                "records": len(df),
            },
        }
    except Exception as e:
        logger.error(f"Error fetching index history for {normalized}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Error fetching index history for {normalized}: {str(e)}",
        )
