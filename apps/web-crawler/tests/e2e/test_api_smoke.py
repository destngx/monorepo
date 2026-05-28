from fastapi.testclient import TestClient

from src.main import app
from src.models.browser import BrowserProbeStep
from src.models.crawl import ThreadCrawlResult
from src.modules.shared.deps import (
    get_browser_probe_service,
    get_thread_crawler_service,
)


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
    async def crawl_thread(
        self,
        url: str,
        cookie: str | None = None,
        timeout_seconds: int = 45,
        max_pages: int | None = None,
        delay_seconds: float = 1.5,
        fresh_profile: bool = False,
        resume_from_page: int | None = None,
    ):
        _ = (
            cookie,
            timeout_seconds,
            max_pages,
            delay_seconds,
            fresh_profile,
            resume_from_page,
        )
        return ThreadCrawlResult(
            thread_id="123",
            url=url,
            output_path="./data/crawls/123.json",
            checkpoint_path="./data/crawls/123.checkpoint.json",
            pages_requested=1,
            pages_crawled=1,
            posts_written=1,
            last_page=1,
            resume_from_page=1,
            stopped_reason="",
            pages=[],
        )


def test_thread_crawl_smoke():
    app.dependency_overrides[get_thread_crawler_service] = (
        lambda: FakeThreadCrawlerService()
    )
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
            assert response.status_code == 200
            assert response.json()["result"]["posts_written"] == 1
    finally:
        app.dependency_overrides.clear()
