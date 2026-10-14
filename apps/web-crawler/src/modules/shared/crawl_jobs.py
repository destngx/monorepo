import asyncio
import uuid
from collections.abc import Awaitable, Callable
from datetime import UTC, datetime

from src.app_logging import get_logger
from src.models.crawl import CrawlJob, CrawlProgress, ThreadCrawlResult

logger = get_logger(__name__)

CrawlRunner = Callable[[Callable[[CrawlProgress], None]], Awaitable[ThreadCrawlResult]]


class CrawlJobConflictError(RuntimeError):
    def __init__(self, job: CrawlJob) -> None:
        super().__init__(f"crawl job {job.job_id} is already running for this thread")
        self.job = job


class CrawlJobManager:
    """Runs crawls as in-process background tasks.

    Job state lives in memory and is lost on restart; the on-disk checkpoint
    is what makes a crawl resumable. Only one job may own a given `key` (the
    thread's on-disk state) at a time.
    """

    def __init__(self, max_finished_jobs: int = 100) -> None:
        self.max_finished_jobs = max_finished_jobs
        self._jobs: dict[str, CrawlJob] = {}
        self._tasks: dict[str, asyncio.Task] = {}
        self._active_by_key: dict[str, str] = {}

    def start(self, *, key: str, url: str, run: CrawlRunner) -> CrawlJob:
        active_job_id = self._active_by_key.get(key)
        if active_job_id is not None:
            raise CrawlJobConflictError(self._jobs[active_job_id])

        job = CrawlJob(
            job_id=uuid.uuid4().hex,
            url=url,
            created_at=datetime.now(UTC),
        )
        self._jobs[job.job_id] = job
        self._active_by_key[key] = job.job_id
        self._tasks[job.job_id] = asyncio.create_task(
            self._run(job, key, run), name=f"crawl-job-{job.job_id}"
        )
        return job

    def get(self, job_id: str) -> CrawlJob | None:
        return self._jobs.get(job_id)

    def list(self) -> list[CrawlJob]:
        return sorted(self._jobs.values(), key=lambda job: job.created_at, reverse=True)

    def cancel(self, job_id: str) -> CrawlJob | None:
        job = self._jobs.get(job_id)
        task = self._tasks.get(job_id)
        if job is not None and task is not None:
            task.cancel()
        return job

    async def shutdown(self) -> None:
        tasks = list(self._tasks.values())
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)

    async def _run(self, job: CrawlJob, key: str, run: CrawlRunner) -> None:
        def on_progress(progress: CrawlProgress) -> None:
            job.progress = progress

        try:
            job.result = await run(on_progress)
            job.status = "succeeded"
        except asyncio.CancelledError:
            job.status = "cancelled"
            raise
        except Exception as exc:
            logger.exception("crawl job %s failed for %s", job.job_id, job.url)
            job.status = "failed"
            job.error = str(exc) or type(exc).__name__
        finally:
            job.finished_at = datetime.now(UTC)
            self._tasks.pop(job.job_id, None)
            self._active_by_key.pop(key, None)
            self._prune_finished()

    def _prune_finished(self) -> None:
        finished = [job for job in self.list() if job.finished_at is not None]
        for job in finished[self.max_finished_jobs :]:
            del self._jobs[job.job_id]
