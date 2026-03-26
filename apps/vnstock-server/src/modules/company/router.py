from fastapi import APIRouter, HTTPException, Query
import logging
import pandas as pd
from ...cache import CacheConfig
from ..shared.deps import get_vnstock_client, rate_limit, cached_response

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/company", tags=["Company"])

@router.get("/overview")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_overview(symbol: str = Query(..., description="Ticker symbol")):
    """Get overview for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.overview()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/profile")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_profile(symbol: str = Query(..., description="Ticker symbol")):
    """Get profile for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.profile()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/ratio-summary")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_ratio_summary(symbol: str = Query(..., description="Ticker symbol")):
    """Get ratio summary for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.ratio_summary()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/trading-stats")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_trading_stats(symbol: str = Query(..., description="Ticker symbol")):
    """Get trading statistics for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.trading_stats()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/news")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_news(
    symbol: str = Query(..., description="Ticker symbol"),
    page_size: int = Query(20, description="Number of items per page"),
):
    """Get news for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.news(page_size=page_size)
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/events")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_events(symbol: str = Query(..., description="Ticker symbol")):
    """Get events for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.events()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/dividends")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_dividends(symbol: str = Query(..., description="Ticker symbol")):
    """Get dividend history for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.dividends()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/officers")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_officers(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of officers for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.officers()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/shareholders")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_shareholders(symbol: str = Query(..., description="Ticker symbol")):
    """Get main shareholders for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.shareholders()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/insider-deals")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_insider_deals(symbol: str = Query(..., description="Ticker symbol")):
    """Get insider trading deals for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.insider_deals()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/subsidiaries")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_subsidiaries(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of subsidiaries for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.subsidiaries()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/affiliate")
@rate_limit()
@cached_response("company", ttl=CacheConfig.COMPANY_TTL)
async def get_company_affiliate(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of affiliate companies for a company."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol=symbol).company.affiliate()
        df_cleaned = df.astype(object).where(pd.notnull(df), None)
        return {"success": True, "data": df_cleaned.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
