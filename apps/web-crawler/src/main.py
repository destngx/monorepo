from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse

from src.app_logging import get_logger, setup_logging
from src.config import WebCrawlerConfig
from src.modules.shared.deps import init_services
from src.routers.browser import router as browser_router
from src.routers.crawl import router as crawl_router
from src.routers.health import router as health_router

setup_logging(debug=WebCrawlerConfig.DEBUG)
logger = get_logger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    WebCrawlerConfig.validate()
    init_services()
    logger.info("web-crawler services initialized")
    yield


app = FastAPI(
    title="Web Crawler",
    description="Python FastAPI browser probe and crawl service for website acquisition experiments",
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
    openapi_url="/openapi.json",
    lifespan=lifespan,
    openapi_tags=[
        {"name": "Health", "description": "Service health and bootstrap checks"},
        {"name": "Browser", "description": "Chromium-backed browser probe endpoint"},
        {"name": "Crawler", "description": "Browser-backed website crawl endpoint"},
    ],
)

app.include_router(health_router)
app.include_router(browser_router)
app.include_router(crawl_router)


@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, exc):
    details = []
    for error in exc.errors():
        details.append(
            {
                "field": ".".join(str(x) for x in error["loc"][1:]),
                "message": error["msg"],
                "type": error["type"],
            }
        )

    return JSONResponse(
        status_code=422,
        content={
            "error": "ValidationError",
            "message": "Request validation failed",
            "details": details,
            "status_code": 422,
        },
    )
