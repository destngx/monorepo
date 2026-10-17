"""
/api/v2: market data by capability, never by source (MARKET_PLATFORM.md
section 3). Every answer is `{data, meta}`; meta says which source served
it, what failed before it, and whether it is a stale last-good copy.
"""

from typing import List, Optional

from fastapi import APIRouter, Depends, FastAPI, Query, Request, Response
from fastapi.exceptions import RequestValidationError
from fastapi.exception_handlers import request_validation_exception_handler
from fastapi.responses import JSONResponse

from ..sources.base import capabilities_of
from .errors import MarketError, NotFound, UpstreamError
from .runtime import get_market_service
from .service import Answer, MarketService

PREFIX = "/api/v2"
STALE_HEADER = "X-Data-Stale"

router = APIRouter(prefix=PREFIX, tags=["Market v2"])

SkipCache = Query(False, description="Bypass the cached answer (the fresh one is still cached)")


def envelope(answer: Answer, response: Response) -> dict:
    if answer.stale:
        # The data time, so consumers can pass it straight to their own clients.
        response.headers[STALE_HEADER] = answer.as_of or "true"
    return {
        "data": answer.data,
        "meta": {
            "provider": answer.provider,
            "fallback": answer.fallback,
            "cached": answer.cached,
            "stale": answer.stale,
            "as_of": answer.as_of,
        },
    }


def _error(status: int, code: str, message: str) -> JSONResponse:
    return JSONResponse(status_code=status, content={"error": {"code": code, "message": message}})


def install_error_handlers(app: FastAPI) -> None:
    @app.exception_handler(MarketError)
    async def market_error(_: Request, exc: MarketError):
        return _error(exc.status, exc.code, str(exc))

    @app.exception_handler(RequestValidationError)
    async def request_error(request: Request, exc: RequestValidationError):
        # v1 keeps FastAPI's 422 shape; v2 uses its own envelope and 400.
        if not request.url.path.startswith(PREFIX):
            return await request_validation_exception_handler(request, exc)
        message = "; ".join(f"{'.'.join(str(p) for p in e['loc'][1:])}: {e['msg']}" for e in exc.errors())
        return _error(400, "validation", message)


@router.get("/ticker")
async def ticker(
    response: Response,
    symbol: str = Query(..., description="Ticker, fund code, gold symbol (SJC, XAU) or coin"),
    type: str = Query(..., description="equity, ifc, gold or crypto"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.ticker(symbol, type, skip_cache), response)


@router.get("/fx")
async def fx(
    response: Response,
    base: str = Query(..., alias="from"),
    quote: str = Query(..., alias="to"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.exchange_rate(base, quote, skip_cache), response)


@router.get("/fx/history")
async def fx_history(
    response: Response,
    pair: str = Query("USD_VND"),
    start: Optional[str] = Query(None, alias="from", description="YYYY-MM-DD, default 30 days ago"),
    end: Optional[str] = Query(None, alias="to", description="YYYY-MM-DD, default today (ICT)"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.fx_history(pair, start, end, skip_cache), response)


@router.get("/series")
async def series(
    response: Response,
    symbol: str = Query(...),
    type: str = Query(..., description="equity, gold_usd or forex"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.price_series(symbol, type, skip_cache), response)


@router.get("/candles")
async def candles(
    response: Response,
    symbol: str = Query(...),
    timeframe: str = Query("1d", description="1h, 4h, 1d or 1w"),
    days: Optional[int] = Query(None, description="Lookback ending now; default per timeframe, at most 3650"),
    start: Optional[str] = Query(None, alias="from", description="YYYY-MM-DD in ICT, instead of days"),
    end: Optional[str] = Query(None, alias="to", description="YYYY-MM-DD in ICT, default today"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.candles(symbol, timeframe, days, start, end, skip_cache), response)


@router.get("/bank-rates")
async def bank_rates(
    response: Response,
    tenor_months: Optional[int] = Query(None, description="Only this tenor in months; 0 is no-term (withdraw any time)"),
    max_tenor_months: Optional[int] = Query(None, description="Only tenors up to this many months, no-term included"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    """Savings rates by bank and tenor. A tenor filter skips sources without tenors (fmarket's headline rates)."""
    return envelope(await market.bank_rates(skip_cache, tenor_months=tenor_months, max_tenor_months=max_tenor_months), response)


@router.get("/funds")
async def funds(
    response: Response,
    asset_type: List[str] = Query(["STOCK"], description="STOCK, BOND, BALANCED, ...; repeatable"),
    mmf: bool = Query(False, description="Money-market funds only (asset types are ignored)"),
    page: int = Query(1),
    page_size: int = Query(10),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.funds(asset_type, mmf, page, page_size, skip_cache), response)


# Static paths before /funds/{code}.
@router.get("/funds/issuers")
async def fund_issuers(response: Response, skip_cache: bool = SkipCache, market: MarketService = Depends(get_market_service)):
    return envelope(await market.issuers(skip_cache), response)


@router.get("/funds/{code}")
async def fund(code: str, response: Response, skip_cache: bool = SkipCache, market: MarketService = Depends(get_market_service)):
    return envelope(await market.fund(code, skip_cache), response)


@router.get("/funds/{fund}/nav")
async def fund_nav(fund: str, response: Response, skip_cache: bool = SkipCache, market: MarketService = Depends(get_market_service)):
    """Full NAV history; `fund` is the fmarket product id or the fund code."""
    return envelope(await market.fund_nav(fund, skip_cache), response)


@router.get("/gold/products")
async def gold_products(response: Response, skip_cache: bool = SkipCache, market: MarketService = Depends(get_market_service)):
    return envelope(await market.gold_products(skip_cache), response)


@router.get("/gold/history")
async def gold_history(
    response: Response,
    start: Optional[str] = Query(None, alias="from", description="YYYY-MM-DD, default 30 days ago"),
    end: Optional[str] = Query(None, alias="to", description="YYYY-MM-DD, default today (ICT)"),
    skip_cache: bool = SkipCache,
    market: MarketService = Depends(get_market_service),
):
    return envelope(await market.gold_history(start, end, skip_cache), response)


@router.get("/providers")
async def providers(market: MarketService = Depends(get_market_service)):
    return {"data": [{"name": s.name, "capabilities": capabilities_of(s)} for s in market.sources.values()]}


@router.get("/providers/{name}/health")
async def provider_health(name: str, market: MarketService = Depends(get_market_service)):
    source = market.sources.get(name.strip().lower())
    if source is None:
        raise NotFound(f"unknown market provider: {name}")
    try:
        status = await source.health()
    except Exception as e:
        raise UpstreamError(f"{source.name} health check failed: {str(e) or type(e).__name__}") from e
    return {"data": {"provider": source.name, "status": status}}
