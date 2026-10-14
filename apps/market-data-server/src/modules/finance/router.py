from fastapi import APIRouter, HTTPException, Query
import logging
from ...cache import CacheConfig
from ..shared.deps import finance_client, upstream_endpoint
from ..shared.frames import to_records

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/finance", tags=["Finance"])

PERIOD_ALIASES = {
    "quarter": "quarter", "quarterly": "quarter", "q": "quarter",
    "year": "year", "yearly": "year", "annual": "year", "y": "year",
}
STATEMENTS = ("balance_sheet", "income_statement", "cash_flow")
PERIOD_QUERY = Query("quarter", description="Period: quarter | year (QUARTERLY / ANNUAL also accepted)")


def normalize_period(period: str) -> str:
    value = PERIOD_ALIASES.get((period or "").strip().lower())
    if value is None:
        raise HTTPException(
            status_code=400,
            detail=f"Invalid period '{period}'. Use quarter or year.",
        )
    return value


def _statement(symbol: str, period: str, method: str):
    period = normalize_period(period)
    try:
        client = finance_client(symbol.strip().upper(), period)
        data = getattr(client, method)(period=period)
        return {"success": True, "data": to_records(data), "period": period, "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching finance.{method} for {symbol}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/balance-sheet")
@upstream_endpoint("finance-balance-sheet", ttl=CacheConfig.FINANCE_TTL)
def get_balance_sheet(symbol: str = Query(..., description="Ticker symbol"), period: str = PERIOD_QUERY):
    """Get balance sheet data for a symbol."""
    return _statement(symbol, period, "balance_sheet")


@router.get("/income-statement")
@upstream_endpoint("finance-income-statement", ttl=CacheConfig.FINANCE_TTL)
def get_income_statement(symbol: str = Query(..., description="Ticker symbol"), period: str = PERIOD_QUERY):
    """Get income statement data for a symbol."""
    return _statement(symbol, period, "income_statement")


@router.get("/cash-flow")
@upstream_endpoint("finance-cash-flow", ttl=CacheConfig.FINANCE_TTL)
def get_cash_flow(symbol: str = Query(..., description="Ticker symbol"), period: str = PERIOD_QUERY):
    """Get cash flow data for a symbol."""
    return _statement(symbol, period, "cash_flow")


@router.get("/ratio")
@upstream_endpoint("finance-ratio", ttl=CacheConfig.FINANCE_TTL)
def get_ratio(symbol: str = Query(..., description="Ticker symbol"), period: str = PERIOD_QUERY):
    """Get financial ratios for a symbol."""
    return _statement(symbol, period, "ratio")


@router.get("/get-all")
@upstream_endpoint("finance-get-all", ttl=CacheConfig.FINANCE_TTL, cost=len(STATEMENTS))
def get_finance_all(symbol: str = Query(..., description="Ticker symbol"), period: str = PERIOD_QUERY):
    """Get balance sheet, income statement and cash flow in one response (3 upstream calls)."""
    period = normalize_period(period)
    try:
        client = finance_client(symbol.strip().upper(), period)
        data = {name: to_records(getattr(client, name)(period=period)) for name in STATEMENTS}
        return {"success": True, "data": data, "period": period, "cached": False}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching all statements for {symbol}: {e}")
        raise HTTPException(status_code=500, detail=str(e))
