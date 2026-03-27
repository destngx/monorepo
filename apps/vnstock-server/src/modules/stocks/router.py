from fastapi import APIRouter, HTTPException, Query
import logging
import pandas as pd
from datetime import datetime, timedelta
from typing import Optional
from ...config import ServerConfig
from ...cache import CacheConfig
from ..shared.deps import get_vnstock_client, is_authenticated, rate_limit, cached_response

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/stocks", tags=["Stocks"])

@router.get("/quote")
@rate_limit()
@cached_response("quote", ttl=CacheConfig.QUOTE_TTL)
async def get_quote(symbol: str = Query(..., description="Ticker symbol")):
    """Get the latest price and metadata for a specific symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol, source=ServerConfig.DEFAULT_SOURCE).quote.history(
            start=(datetime.now() - timedelta(days=7)).strftime("%Y-%m-%d"),
            end=datetime.now().strftime("%Y-%m-%d"),
        )

        if df.empty:
            raise HTTPException(status_code=404, detail=f"No data found for symbol {symbol}")

        latest = df.iloc[-1].to_dict()
        if "time" in latest and isinstance(latest["time"], (pd.Timestamp, datetime)):
            latest["time"] = latest["time"].strftime("%Y-%m-%d %H:%M:%S")

        return {
            "success": True,
            "data": {
                "symbol": symbol,
                "price": latest.get("close"),
                "change": latest.get("change", 0),
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
        raise HTTPException(status_code=500, detail=f"Error fetching quote for {symbol}: {str(e)}")

@router.get("/intraday")
@rate_limit()
@cached_response("intraday", ttl=CacheConfig.INTRADAY_TTL)
async def get_intraday_data(
    symbol: str = Query(..., description="Ticker symbol"),
    page_size: int = Query(100, description="Number of records per page"),
):
    """Get intraday trading data."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).quote.intraday(page_size=page_size)
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/price-depth")
@rate_limit()
@cached_response("price-depth", ttl=CacheConfig.QUOTE_TTL)
async def get_price_depth(symbol: str = Query(..., description="Ticker symbol")):
    """Get real-time price depth (Level 2 data) for a symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).quote.price_depth()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/price-board")
@rate_limit()
@cached_response("price-board", ttl=CacheConfig.PRICE_BOARD_TTL)
async def get_price_board(symbols: str = Query(..., description="Comma-separated ticker symbols")):
    """Get price board for multiple symbols."""
    try:
        symbols_list = [s.strip() for s in symbols.split(",")]
        client = get_vnstock_client()
        df = client.stock(symbol=symbols_list[0]).trading.price_board(symbols_list=symbols_list)
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/historical")
@rate_limit()
@cached_response("historical", ttl=CacheConfig.HISTORICAL_TTL)
async def get_historical_data(
    symbol: str = Query(..., description="Ticker symbol (e.g., GMD, VCB)"),
    start_date: Optional[str] = Query(None, description="Start date in YYYY-MM-DD format"),
    end_date: Optional[str] = Query(None, description="End date in YYYY-MM-DD format"),
    resolution: str = Query("1D", description="Data resolution (1M, 5M, 15M, 30M, 1H, 1D)"),
):
    """Get historical stock data for a specific symbol."""
    if not end_date:
        end_date = datetime.now().strftime("%Y-%m-%d")
    if not start_date:
        start_date = (datetime.now() - timedelta(days=30)).strftime("%Y-%m-%d")

    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol, source=ServerConfig.DEFAULT_SOURCE).quote.history(start=start_date, end=end_date)

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
        }
    except Exception as e:
        logger.error(f"Error fetching historical data for {symbol}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error fetching historical data for {symbol}: {str(e)}")

@router.get("/index-history")
@rate_limit()
@cached_response("index-history", ttl=CacheConfig.HISTORICAL_TTL)
async def get_index_history(
    symbol: str = Query(..., description="Index symbol (e.g., VNINDEX, HNX, VN30, UPCOM)"),
    start_date: Optional[str] = Query(None, description="Start date in YYYY-MM-DD format"),
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
        client = get_vnstock_client()
        df = client.stock(symbol=normalized, source=ServerConfig.DEFAULT_SOURCE).quote.history(
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
        raise HTTPException(status_code=500, detail=f"Error fetching index history for {normalized}: {str(e)}")
