from fastapi import APIRouter, Depends, HTTPException, status

from src.config import WebCrawlerConfig
from src.models.crawl import (
    CrawlJob,
    CrawlJobListResponse,
    CrawlJobResponse,
    ThreadCrawlRequest,
)
from src.modules.shared.crawl_jobs import CrawlJobConflictError, CrawlJobManager
from src.modules.shared.deps import get_crawl_job_manager, get_thread_crawler_service
from src.modules.sites.voz.crawler import VozThreadCrawler

router = APIRouter(tags=["Crawler"])


@router.post(
    "/v1/threads/crawl/browser",
    response_model=CrawlJobResponse,
    status_code=status.HTTP_202_ACCEPTED,
)
async def crawl_thread_browser(
    request: ThreadCrawlRequest,
    service: VozThreadCrawler = Depends(get_thread_crawler_service),
    jobs: CrawlJobManager = Depends(get_crawl_job_manager),
) -> CrawlJobResponse:
    def run(on_progress):
        return service.crawl_thread(
            url=request.url,
            cookie=request.cookie,
            timeout_seconds=request.timeout_seconds
            or WebCrawlerConfig.BROWSER_TIMEOUT_SECONDS,
            max_pages=request.max_pages,
            delay_seconds=(
                5.0 if request.delay_seconds is None else request.delay_seconds
            ),
            fresh_profile=request.fresh_profile,
            resume_from_page=request.resume_from_page,
            restart=request.restart,
            on_progress=on_progress,
        )

    try:
        job = jobs.start(key=service.thread_key(request.url), url=request.url, run=run)
    except CrawlJobConflictError as exc:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail={"message": str(exc), "job_id": exc.job.job_id},
        ) from exc
    return CrawlJobResponse(job=job)


@router.get("/v1/threads/crawl/jobs", response_model=CrawlJobListResponse)
async def list_crawl_jobs(
    jobs: CrawlJobManager = Depends(get_crawl_job_manager),
) -> CrawlJobListResponse:
    return CrawlJobListResponse(jobs=jobs.list())


@router.get("/v1/threads/crawl/jobs/{job_id}", response_model=CrawlJobResponse)
async def get_crawl_job(
    job_id: str,
    jobs: CrawlJobManager = Depends(get_crawl_job_manager),
) -> CrawlJobResponse:
    return CrawlJobResponse(job=_require_job(jobs.get(job_id), job_id))


@router.delete(
    "/v1/threads/crawl/jobs/{job_id}",
    response_model=CrawlJobResponse,
    status_code=status.HTTP_202_ACCEPTED,
)
async def cancel_crawl_job(
    job_id: str,
    jobs: CrawlJobManager = Depends(get_crawl_job_manager),
) -> CrawlJobResponse:
    return CrawlJobResponse(job=_require_job(jobs.cancel(job_id), job_id))


def _require_job(job: CrawlJob | None, job_id: str) -> CrawlJob:
    if job is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"crawl job {job_id} not found",
        )
    return job
