from fastapi import APIRouter

from src.config import WebCrawlerConfig
from src.models.health import HealthResponse

router = APIRouter(tags=["Health"])


@router.get("/health", response_model=HealthResponse)
async def health() -> HealthResponse:
    return HealthResponse(
        status="ok",
        service="web-crawler",
        browser_configured=bool(WebCrawlerConfig.CHROME_BIN),
    )
