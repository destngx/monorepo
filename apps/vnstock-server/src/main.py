"""
FastAPI wrapper for vnstock using Vertical Slice Architecture.
"""

from fastapi import FastAPI
import logging
from datetime import datetime

from .config import VnstockConfig
from .modules.shared.deps import init_services, get_cache_manager, get_rate_limiter, is_authenticated
from .modules.listing.router import router as listing_router
from .modules.stocks.router import router as stocks_router
from .modules.finance.router import router as finance_router
from .modules.company.router import router as company_router
from .modules.health.router import router as health_router

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Vnstock API Wrapper",
    description="A FastAPI wrapper for the vnstock library with API key authentication for better rate limits and reliability.",
    version="1.0.0",
)

# Initialize shared services
init_services()

# Include routers
app.include_router(listing_router)
app.include_router(stocks_router)
app.include_router(finance_router)
app.include_router(company_router)
app.include_router(health_router)

@app.on_event("startup")
async def startup_event():
    try:
        VnstockConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    cache_manager = get_cache_manager()
    if cache_manager:
        if cache_manager.enabled:
            logger.info("Cache layer is operational")
        else:
            logger.warning("Cache layer is disabled")

    rate_limiter = get_rate_limiter()
    if rate_limiter:
        status = rate_limiter.get_status()
        logger.info(
            f"✓ Rate limiter ready: {status.tier} ({status.requests_per_minute} req/min, {status.requests_per_day} req/day)"
        )

@app.get("/")
def read_root():
    return {
        "message": "Welcome to Vnstock API Wrapper. Access /docs for API documentation.",
        "version": "1.0.0",
        "status": "ready",
        "authenticated": is_authenticated()
    }

@app.get("/health")
def health_check_legacy():
    """Legacy health check compatible with old client versions."""
    return {
        "status": "ok",
        "service": "vnstock-server",
        "timestamp": datetime.now().isoformat(),
        "authenticated": is_authenticated(),
        "note": "Use /api/v1/health for detailed status"
    }
