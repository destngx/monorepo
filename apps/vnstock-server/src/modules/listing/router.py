from fastapi import APIRouter, HTTPException, Query
import logging
import pandas as pd
from ...config import ServerConfig
from ...cache import CacheConfig
from ..shared.deps import get_vnstock_client, is_authenticated, rate_limit, cached_response

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/listing", tags=["Listing"])

@router.get("/stocks")
@rate_limit()
@cached_response("listing", ttl=CacheConfig.LISTING_TTL)
async def get_listing():
    """Get a list of all companies listed on the Vietnamese stock market."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND", source=ServerConfig.DEFAULT_SOURCE).listing.all_symbols()
        return {
            "success": True,
            "data": df.to_dict(orient="records"),
            "source": ServerConfig.DEFAULT_SOURCE,
            "authenticated": is_authenticated(),
            "cached": False,
        }
    except Exception as e:
        logger.error(f"Error fetching stock listing: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error fetching stock listing: {str(e)}")

@router.get("/bonds")
@rate_limit()
@cached_response("listing-bonds", ttl=CacheConfig.LISTING_TTL)
async def get_all_bonds():
    """Get all corporate bonds."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.all_bonds()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/covered-warrants")
@rate_limit()
@cached_response("listing-covered-warrants", ttl=CacheConfig.LISTING_TTL)
async def get_all_covered_warrants():
    """Get all covered warrants."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.all_covered_warrant()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/future-indices")
@rate_limit()
@cached_response("listing-future-indices", ttl=CacheConfig.LISTING_TTL)
async def get_all_future_indices():
    """Get all future indices."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.all_future_indices()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/government-bonds")
@rate_limit()
@cached_response("listing-government-bonds", ttl=CacheConfig.LISTING_TTL)
async def get_all_government_bonds():
    """Get all government bonds."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.all_government_bonds()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/industries-icb")
@rate_limit()
@cached_response("listing-industries-icb", ttl=CacheConfig.LISTING_TTL)
async def get_industries_icb():
    """Get ICB industry classification."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.industries_icb()
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/symbols-by-exchange")
@rate_limit()
@cached_response("listing-exchange", ttl=CacheConfig.LISTING_TTL)
async def get_symbols_by_exchange(exchange: str = Query(..., description="Exchange code (HOSE, HNX, UPCOM)")):
    """Get symbols by exchange."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.symbols_by_exchange(exchange=exchange)
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/symbols-by-group")
@rate_limit()
@cached_response("listing-group", ttl=CacheConfig.LISTING_TTL)
async def get_symbols_by_group(group: str = Query(..., description="Symbol group")):
    """Get symbols by group."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.symbols_by_group(group=group)
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/symbols-by-industries")
@rate_limit()
@cached_response("listing-industry", ttl=CacheConfig.LISTING_TTL)
async def get_symbols_by_industries(industry: str = Query(..., description="Industry name/code")):
    """Get symbols by industry."""
    try:
        client = get_vnstock_client()
        df = client.stock(symbol="VND").listing.symbols_by_industries(industry_name=industry)
        return {"success": True, "data": df.to_dict(orient="records"), "cached": False}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
