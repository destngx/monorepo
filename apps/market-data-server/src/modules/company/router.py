from fastapi import APIRouter, HTTPException, Query
import logging
from typing import Optional
from ...cache import CacheConfig
from ..shared.deps import company_client, upstream_endpoint
from ..shared.frames import to_records

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/company", tags=["Company"])

# Methods only one provider implements in vnstock 4.x.
VCI = "VCI"
KBS = "KBS"


def _company(symbol: str, method: str, source: Optional[str] = None, **kwargs):
    try:
        data = getattr(company_client(symbol.strip().upper(), source), method)(**kwargs)
        return {"success": True, "data": to_records(data), "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching company.{method} for {symbol}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/overview")
@upstream_endpoint("company-overview", ttl=CacheConfig.COMPANY_TTL)
def get_company_overview(symbol: str = Query(..., description="Ticker symbol")):
    """Get overview for a company."""
    return _company(symbol, "overview")


@router.get("/ratio-summary")
@upstream_endpoint("company-ratio-summary", ttl=CacheConfig.COMPANY_TTL)
def get_company_ratio_summary(symbol: str = Query(..., description="Ticker symbol")):
    """Get ratio summary for a company (VCI)."""
    return _company(symbol, "ratio_summary", source=VCI)


@router.get("/trading-stats")
@upstream_endpoint("company-trading-stats", ttl=CacheConfig.COMPANY_TTL)
def get_company_trading_stats(symbol: str = Query(..., description="Ticker symbol")):
    """Get trading statistics for a company (VCI)."""
    return _company(symbol, "trading_stats", source=VCI)


@router.get("/news")
@upstream_endpoint("company-news", ttl=CacheConfig.COMPANY_TTL)
def get_company_news(
    symbol: str = Query(..., description="Ticker symbol"),
    page_size: int = Query(20, ge=1, le=200, description="Number of items to return"),
):
    """Get news for a company."""
    result = _company(symbol, "news")
    # Not every provider pages news, so trim here.
    result["data"] = result["data"][:page_size]
    return result


@router.get("/events")
@upstream_endpoint("company-events", ttl=CacheConfig.COMPANY_TTL)
def get_company_events(symbol: str = Query(..., description="Ticker symbol")):
    """Get events for a company."""
    return _company(symbol, "events")


@router.get("/officers")
@upstream_endpoint("company-officers", ttl=CacheConfig.COMPANY_TTL)
def get_company_officers(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of officers for a company."""
    return _company(symbol, "officers")


@router.get("/shareholders")
@upstream_endpoint("company-shareholders", ttl=CacheConfig.COMPANY_TTL)
def get_company_shareholders(symbol: str = Query(..., description="Ticker symbol")):
    """Get main shareholders for a company."""
    return _company(symbol, "shareholders")


@router.get("/ownership")
@upstream_endpoint("company-ownership", ttl=CacheConfig.COMPANY_TTL)
def get_company_ownership(symbol: str = Query(..., description="Ticker symbol")):
    """Get ownership structure by owner type (KBS)."""
    return _company(symbol, "ownership", source=KBS)


@router.get("/capital-history")
@upstream_endpoint("company-capital-history", ttl=CacheConfig.COMPANY_TTL)
def get_company_capital_history(symbol: str = Query(..., description="Ticker symbol")):
    """Get charter capital history (KBS)."""
    return _company(symbol, "capital_history", source=KBS)


@router.get("/insider-deals")
@upstream_endpoint("company-insider-deals", ttl=CacheConfig.COMPANY_TTL)
def get_company_insider_deals(
    symbol: str = Query(..., description="Ticker symbol"),
    page_size: int = Query(20, ge=1, le=100, description="Number of items to return"),
):
    """Get insider trading deals for a company (KBS)."""
    return _company(symbol, "insider_trading", source=KBS, page_size=page_size)


@router.get("/subsidiaries")
@upstream_endpoint("company-subsidiaries", ttl=CacheConfig.COMPANY_TTL)
def get_company_subsidiaries(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of subsidiaries for a company."""
    return _company(symbol, "subsidiaries")


@router.get("/affiliate")
@upstream_endpoint("company-affiliate", ttl=CacheConfig.COMPANY_TTL)
def get_company_affiliate(symbol: str = Query(..., description="Ticker symbol")):
    """Get list of affiliate companies for a company."""
    return _company(symbol, "affiliate")
