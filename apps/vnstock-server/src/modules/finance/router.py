from fastapi import APIRouter, HTTPException, Query
import logging
import pandas as pd
from ...cache import CacheConfig
from ..shared.deps import get_vnstock_client, rate_limit, cached_response

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/finance", tags=["Finance"])

@router.get("/balance-sheet")
@rate_limit()
@cached_response("finance", ttl=CacheConfig.FINANCE_TTL)
async def get_balance_sheet(
    symbol: str = Query(..., description="Ticker symbol"),
    period: str = Query("QUARTERLY", description="Period: ANNUAL | QUARTERLY"),
):
    """Get balance sheet data for a symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).finance.balance_sheet(period=period)
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/income-statement")
@rate_limit()
@cached_response("finance", ttl=CacheConfig.FINANCE_TTL)
async def get_income_statement(
    symbol: str = Query(..., description="Ticker symbol"),
    period: str = Query("QUARTERLY", description="Period: ANNUAL | QUARTERLY"),
):
    """Get income statement data for a symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).finance.income_statement(period=period)
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/cash-flow")
@rate_limit()
@cached_response("finance", ttl=CacheConfig.FINANCE_TTL)
async def get_cash_flow(
    symbol: str = Query(..., description="Ticker symbol"),
    period: str = Query("QUARTERLY", description="Period: ANNUAL | QUARTERLY"),
):
    """Get cash flow data for a symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).finance.cash_flow(period=period)
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/ratio")
@rate_limit()
@cached_response("finance", ttl=CacheConfig.FINANCE_TTL)
async def get_ratio(
    symbol: str = Query(..., description="Ticker symbol"),
    period: str = Query("QUARTERLY", description="Period: ANNUAL | QUARTERLY"),
):
    """Get financial ratios for a symbol."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).finance.ratio(period=period)
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/get-all")
@rate_limit()
@cached_response("finance", ttl=CacheConfig.FINANCE_TTL)
async def get_finance_all(
    symbol: str = Query(..., description="Ticker symbol"),
    period: str = Query("QUARTERLY", description="Period: ANNUAL | QUARTERLY"),
):
    """Get all financial statements for a symbol."""
    try:
        client = get_vnstock_client()
        stock = client.stock(symbol=symbol)
        if hasattr(stock.finance, "get_all") and callable(stock.finance.get_all):
            df = stock.finance.get_all(period=period)
        else:
            raise HTTPException(status_code=501, detail="finance.get_all() is not callable in this vnstock version")
            
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
