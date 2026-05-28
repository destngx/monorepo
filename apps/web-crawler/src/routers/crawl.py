from fastapi import APIRouter, Depends

from src.config import WebCrawlerConfig
from src.models.crawl import ThreadCrawlRequest, ThreadCrawlResponse
from src.modules.shared.deps import get_thread_crawler_service
from src.modules.sites.voz.crawler import VozThreadCrawler

router = APIRouter(tags=["Crawler"])


@router.post("/v1/threads/crawl/browser", response_model=ThreadCrawlResponse)
async def crawl_thread_browser(
    request: ThreadCrawlRequest,
    service: VozThreadCrawler = Depends(get_thread_crawler_service),
) -> ThreadCrawlResponse:
    result = await service.crawl_thread(
        url=request.url,
        cookie=request.cookie,
        timeout_seconds=request.timeout_seconds or WebCrawlerConfig.BROWSER_TIMEOUT_SECONDS,
        max_pages=request.max_pages,
        delay_seconds=5.0 if request.delay_seconds is None else request.delay_seconds,
        fresh_profile=request.fresh_profile,
        resume_from_page=request.resume_from_page,
    )
    return ThreadCrawlResponse(result=result)
