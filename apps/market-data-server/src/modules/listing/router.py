from fastapi import APIRouter, HTTPException, Query
import logging
from typing import Optional
from ...config import ServerConfig
from ...cache import CacheConfig
from ..shared.deps import listing_client, is_authenticated, upstream_endpoint
from ..shared.frames import to_records

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/listing", tags=["Listing"])

# KBS does not implement ICB data or government bonds.
VCI = "VCI"
EXCHANGE_ALIASES = {"HOSE": "HSX"}


def _listing(method: str, source: Optional[str] = None, **kwargs):
    try:
        data = getattr(listing_client(source), method)(**kwargs)
        return data
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching listing.{method}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


def _ok(data):
    return {"success": True, "data": to_records(data), "cached": False}


@router.get("/stocks")
@upstream_endpoint("listing-stocks", ttl=CacheConfig.LISTING_TTL)
def get_listing():
    """Get a list of all companies listed on the Vietnamese stock market."""
    result = _ok(_listing("all_symbols"))
    result.update(source=ServerConfig.DEFAULT_SOURCE, authenticated=is_authenticated())
    return result


@router.get("/bonds")
@upstream_endpoint("listing-bonds", ttl=CacheConfig.LISTING_TTL)
def get_all_bonds():
    """Get all corporate bonds."""
    return _ok(_listing("all_bonds"))


@router.get("/covered-warrants")
@upstream_endpoint("listing-covered-warrants", ttl=CacheConfig.LISTING_TTL)
def get_all_covered_warrants():
    """Get all covered warrants."""
    return _ok(_listing("all_covered_warrant"))


@router.get("/future-indices")
@upstream_endpoint("listing-future-indices", ttl=CacheConfig.LISTING_TTL)
def get_all_future_indices():
    """Get all future indices."""
    return _ok(_listing("all_future_indices"))


@router.get("/government-bonds")
@upstream_endpoint("listing-government-bonds", ttl=CacheConfig.LISTING_TTL)
def get_all_government_bonds():
    """Get all government bonds (VCI)."""
    return _ok(_listing("all_government_bonds", source=VCI))


@router.get("/industries-icb")
@upstream_endpoint("listing-industries-icb", ttl=CacheConfig.LISTING_TTL)
def get_industries_icb():
    """Get ICB industry classification (VCI)."""
    return _ok(_listing("industries_icb", source=VCI))


@router.get("/symbols-by-exchange")
@upstream_endpoint("listing-symbols-by-exchange", ttl=CacheConfig.LISTING_TTL)
def get_symbols_by_exchange(
    exchange: str = Query(..., description="Exchange code (HOSE, HNX, UPCOM)"),
):
    """Get symbols listed on one exchange."""
    code = exchange.strip().upper()
    code = EXCHANGE_ALIASES.get(code, code)
    df = _listing("symbols_by_exchange")
    if "exchange" not in df.columns:
        raise HTTPException(status_code=502, detail="Upstream listing has no exchange column")
    exchanges = df["exchange"].astype(str).str.upper()
    return _ok(df[exchanges.isin({code, EXCHANGE_ALIASES.get(code, code)})])


@router.get("/symbols-by-group")
@upstream_endpoint("listing-symbols-by-group", ttl=CacheConfig.LISTING_TTL)
def get_symbols_by_group(group: str = Query(..., description="Symbol group, e.g. VN30")):
    """Get symbols by group."""
    return _ok(_listing("symbols_by_group", group=group.strip().upper()))


@router.get("/symbols-by-industries")
@upstream_endpoint("listing-symbols-by-industries", ttl=CacheConfig.LISTING_TTL)
def get_symbols_by_industries(
    industry: str = Query(..., description="ICB code (e.g. 8300) or industry name"),
):
    """Get symbols in one ICB industry (VCI)."""
    df = _listing("symbols_by_industries", source=VCI)
    wanted = industry.strip().lower()
    mask = (df["icb_code"].astype(str) == wanted) | (df["icb_name"].astype(str).str.lower() == wanted)
    return _ok(df[mask])
