import asyncio

import anyio
import pytest

from src.models.crawl import CrawlProgress, ThreadCrawlResult
from src.modules.shared.crawl_jobs import CrawlJobConflictError, CrawlJobManager


async def _settle(manager: CrawlJobManager) -> None:
    while any(job.finished_at is None for job in manager.list()):
        await asyncio.sleep(0)


def _result() -> ThreadCrawlResult:
    return ThreadCrawlResult(
        thread_id="1",
        url="https://voz.vn/t/x.1/",
        output_path="1.json",
        pages_path="1.pages.jsonl",
        checkpoint_path="1.checkpoint.json",
        pages_requested=1,
        pages_crawled=1,
        posts_written=3,
        last_page=1,
        resume_from_page=1,
    )


def test_job_records_progress_and_result():
    async def scenario():
        manager = CrawlJobManager()

        async def run(on_progress):
            on_progress(CrawlProgress(current_page=1, pages_crawled=1, posts_written=3))
            return _result()

        job = manager.start(key="voz.vn/1", url="https://voz.vn/t/x.1/", run=run)
        assert job.status == "running"
        await _settle(manager)
        return manager.get(job.job_id)

    job = anyio.run(scenario)

    assert job.status == "succeeded"
    assert job.progress.posts_written == 3
    assert job.result.posts_written == 3
    assert job.finished_at is not None


def test_job_failure_is_recorded():
    async def scenario():
        manager = CrawlJobManager()

        async def run(on_progress):
            raise RuntimeError("disk full")

        job = manager.start(key="voz.vn/1", url="https://voz.vn/t/x.1/", run=run)
        await _settle(manager)
        return job

    job = anyio.run(scenario)

    assert job.status == "failed"
    assert job.error == "disk full"


def test_second_job_for_same_thread_conflicts_until_first_finishes():
    async def scenario():
        manager = CrawlJobManager()
        release = asyncio.Event()

        async def run(on_progress):
            await release.wait()
            return _result()

        first = manager.start(key="voz.vn/1", url="u", run=run)
        with pytest.raises(CrawlJobConflictError) as exc_info:
            manager.start(key="voz.vn/1", url="u", run=run)
        assert exc_info.value.job.job_id == first.job_id

        other = manager.start(key="voz.vn/2", url="u", run=run)
        release.set()
        await _settle(manager)
        again = manager.start(key="voz.vn/1", url="u", run=run)
        await _settle(manager)
        return first, other, again

    first, other, again = anyio.run(scenario)

    assert first.status == other.status == again.status == "succeeded"


def test_cancel_stops_running_job():
    async def scenario():
        manager = CrawlJobManager()

        async def run(on_progress):
            await asyncio.sleep(60)

        job = manager.start(key="voz.vn/1", url="u", run=run)
        await asyncio.sleep(0)
        manager.cancel(job.job_id)
        await manager.shutdown()
        return job

    job = anyio.run(scenario)

    assert job.status == "cancelled"


def test_finished_jobs_are_pruned_to_limit():
    async def scenario():
        manager = CrawlJobManager(max_finished_jobs=2)

        async def run(on_progress):
            return _result()

        for index in range(4):
            manager.start(key=f"k{index}", url="u", run=run)
            await _settle(manager)
        return manager.list()

    assert len(anyio.run(scenario)) == 2
