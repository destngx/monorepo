"""
Market data server: one API over Vietnamese and global market sources.
"""

from fastapi import FastAPI
import logging
from contextlib import asynccontextmanager
from datetime import datetime
from pathlib import Path

from dotenv import load_dotenv

# Runtime input only. Real environment variables win over the file.
load_dotenv(Path(__file__).resolve().parent.parent / ".env.local", override=False)

from .config import VnstockConfig  # noqa: E402
from .modules.shared.deps import init_services, get_cache_manager, get_rate_limiter, is_authenticated
from .modules.listing.router import router as listing_router
from .modules.stocks.router import router as stocks_router
from .modules.finance.router import router as finance_router
from .modules.company.router import router as company_router
from .modules.health.router import router as health_router
from .market.api import install_error_handlers, router as market_v2_router
from .market.runtime import close_market_service

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize shared services
init_services()


@asynccontextmanager
async def lifespan(_: FastAPI):
    try:
        VnstockConfig.validate()
        logger.info("Configuration validation passed")
    except ValueError as e:
        logger.warning(f"Configuration note: {e}")

    cache_manager = get_cache_manager()
    if cache_manager:
        logger.info(f"Cache layer is operational ({cache_manager.backend})")

    rate_limiter = get_rate_limiter()
    if rate_limiter:
        status = rate_limiter.get_status()
        logger.info(
            f"✓ Rate limiter ready: {status.tier} ({status.requests_per_minute} req/min, {status.requests_per_day} req/day)"
        )

    yield

    await close_market_service()
    if cache_manager:
        await cache_manager.aclose()


app = FastAPI(
    title="Market Data Server",
    description="Market data for Vietnamese and global markets, cached and rate limited per upstream source.",
    version="1.0.0",
    lifespan=lifespan,
)

# Include routers
app.include_router(listing_router)
app.include_router(stocks_router)
app.include_router(finance_router)
app.include_router(company_router)
app.include_router(health_router)
app.include_router(market_v2_router)
install_error_handlers(app)

@app.get("/")
def read_root():
    return {
        "message": "Welcome to Market Data Server. Access /docs for API documentation.",
        "version": "1.0.0",
        "status": "ready",
        "authenticated": is_authenticated()
    }

@app.get("/health")
def health_check_legacy():
    """Legacy health check compatible with old client versions."""
    return {
        "status": "ok",
        "service": "market-data-server",
        "timestamp": datetime.now().isoformat(),
        "authenticated": is_authenticated(),
        "note": "Use /api/v1/health for detailed status"
    }
