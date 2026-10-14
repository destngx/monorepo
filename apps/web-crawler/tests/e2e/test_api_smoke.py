import time
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

from src.config import WebCrawlerConfig
from src.main import app
from src.models.browser import BrowserProbeStep
from src.models.crawl import CrawlProgress, ThreadCrawlResult
from src.modules.shared.crawl_jobs import CrawlJobManager
from src.modules.shared.deps import (
    get_browser_probe_service,
    get_crawl_job_manager,
    get_thread_crawler_service,
)


@pytest.fixture(autouse=True)
def _startup_config(tmp_path):
    # Lifespan startup validates that Chrome and the data dirs exist.
    with patch.multiple(
        WebCrawlerConfig,
        CHROME_BIN=__file__,
        BROWSER_PROFILE_DIR=str(tmp_path / "browser-profile"),
        CRAWL_OUTPUT_DIR=str(tmp_path / "crawls"),
    ):
        yield


class FakeBrowserProbeService:
    async def probe(
        self,
        url: str,
        page_url: str | None = None,
        cookie: str | None = None,
        timeout_seconds: int = 45,
    ):
        _ = cookie
        return BrowserProbeStep(
            name="browser_probe",
            renderer="playwright",
            url=page_url or url,
            final_url=page_url or url,
            status_code=200,
            content_type="text/html",
            kind="ok",
            bytes=2048,
            post_count=1,
            last_page=1,
            error="",
        )


def test_browser_probe_smoke():
    app.dependency_overrides[get_browser_probe_service] = (
        lambda: FakeBrowserProbeService()
    )
    try:
        with TestClient(app) as client:
            response = client.post(
                "/v1/pages/probe/browser",
                json={"url": "https://voz.vn/t/example.123/", "timeout_seconds": 5},
            )
            assert response.status_code == 200
            assert response.json()["step"]["kind"] == "ok"
    finally:
        app.dependency_overrides.clear()


class FakeThreadCrawlerService:
    def thread_key(self, url: str) -> str:
        return url

    async def crawl_thread(
        self,
        url: str,
        cookie: str | None = None,
        timeout_seconds: int = 45,
        max_pages: int | None = None,
        delay_seconds: float = 1.5,
        fresh_profile: bool = False,
        resume_from_page: int | None = None,
        restart: bool = False,
        on_progress=None,
    ):
        _ = (
            cookie,
            timeout_seconds,
            max_pages,
            delay_seconds,
            fresh_profile,
            resume_from_page,
            restart,
        )
        on_progress(CrawlProgress(current_page=70, pages_crawled=1, posts_written=1))
        return ThreadCrawlResult(
            thread_id="123",
            url=url,
            output_path="./data/crawls/123.json",
            pages_path="./data/crawls/123.pages.jsonl",
            checkpoint_path="./data/crawls/123.checkpoint.json",
            pages_requested=1,
            pages_crawled=1,
            posts_written=1,
            last_page=1,
            resume_from_page=1,
            stopped_reason="",
            pages=[],
        )


def _wait_for_job(client: TestClient, job_id: str) -> dict:
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        job = client.get(f"/v1/threads/crawl/jobs/{job_id}").json()["job"]
        if job["status"] != "running":
            return job
        time.sleep(0.01)
    raise AssertionError(f"crawl job {job_id} did not finish")


def test_thread_crawl_smoke():
    app.dependency_overrides[get_thread_crawler_service] = (
        lambda: FakeThreadCrawlerService()
    )
    manager = CrawlJobManager()
    app.dependency_overrides[get_crawl_job_manager] = lambda: manager
    try:
        with TestClient(app) as client:
            response = client.post(
                "/v1/threads/crawl/browser",
                json={
                    "url": "https://voz.vn/t/example.123/",
                    "timeout_seconds": 5,
                    "max_pages": 1,
                    "delay_seconds": 0,
                    "fresh_profile": True,
                    "resume_from_page": 70,
                },
            )
            assert response.status_code == 202
            job_id = response.json()["job"]["job_id"]

            job = _wait_for_job(client, job_id)

            assert job["status"] == "succeeded"
            assert job["progress"]["current_page"] == 70
            assert job["result"]["posts_written"] == 1
            listed = client.get("/v1/threads/crawl/jobs").json()["jobs"]
            assert [item["job_id"] for item in listed] == [job_id]
            assert client.get("/v1/threads/crawl/jobs/missing").status_code == 404
    finally:
        app.dependency_overrides.clear()
