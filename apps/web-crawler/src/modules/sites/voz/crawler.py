import asyncio
import time
from collections.abc import Callable
from pathlib import Path

from src.models.crawl import (
    CrawlCheckpoint,
    CrawledPage,
    CrawlMode,
    CrawlProgress,
    ThreadCrawlResult,
)
from src.modules.shared.crawl_storage import CrawlStorage
from src.services.browser_client import (
    BrowserAcquireError,
    BrowserClient,
    PlaywrightBrowserClient,
)
from src.services.detector import classify_page

from .pacing import CrawlPacing
from .parser import VozParser


class VozThreadCrawler:
    def __init__(
        self,
        browser_client: BrowserClient,
        parser: VozParser | None = None,
        output_dir: str = "./data/crawls",
    ) -> None:
        self.browser_client = browser_client
        self.parser = parser or VozParser()
        self.storage = CrawlStorage(output_dir)
        self.pacing = CrawlPacing()

    def thread_key(self, url: str) -> str:
        """Identify the on-disk state a crawl of `url` owns."""
        thread_id = self.parser.thread_id_from_url(url)
        return str(self.storage.thread_dir(url, thread_id))

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
        on_progress: Callable[[CrawlProgress], None] | None = None,
    ) -> ThreadCrawlResult:
        browser_client = self._browser_client(fresh_profile)
        thread_id = self.parser.thread_id_from_url(url)
        paths = self.storage.resolve_thread_paths(url, thread_id)
        await asyncio.to_thread(self.storage.ensure_thread_dir, paths.thread_dir)
        if restart:
            await asyncio.to_thread(
                self.storage.clear_thread_state,
                paths.output_path,
                paths.pages_path,
                paths.checkpoint_path,
            )
            saved_pages: dict[int, dict] = {}
        else:
            saved_pages = await asyncio.to_thread(
                self.storage.load_pages, paths.pages_path
            )

        mode: CrawlMode
        if resume_from_page is not None:
            mode = "resume"
            resume_from_page = max(1, resume_from_page)
        elif saved_pages:
            mode = "update"
            resume_from_page = self._update_start_page(saved_pages)
        else:
            mode = "fresh"
            resume_from_page = 1

        last_page = 1
        pages: list[CrawledPage] = []
        posts_written = self._posts_written(saved_pages)
        pages_requested = resume_from_page
        stopped_reason = ""
        retried_with_fresh_profile = fresh_profile

        current_page = resume_from_page

        def checkpoint(last_successful_page: int) -> CrawlCheckpoint:
            return CrawlCheckpoint.for_page(
                thread_id=thread_id,
                url=url,
                last_successful_page=last_successful_page,
                last_page=last_page,
                pages_requested=max(pages_requested, last_page),
                posts_written=posts_written,
                output_path=str(paths.output_path),
            )

        try:
            while True:
                if delay_seconds > 0 and pages:
                    await asyncio.sleep(self.pacing.delay_seconds(delay_seconds))

                current_url = (
                    url
                    if current_page == 1
                    else self.parser.page_url(url, current_page)
                )
                crawled_page, replies = await self._fetch_page(
                    url=current_url,
                    page=current_page,
                    thread_id=thread_id,
                    cookie=cookie,
                    timeout_seconds=timeout_seconds,
                    browser_client=browser_client,
                    thread_dir=paths.thread_dir,
                )
                pages.append(crawled_page)
                last_page = max(last_page, crawled_page.last_page)
                pages_requested = self._pages_requested(last_page, max_pages)

                if crawled_page.kind == "ok":
                    page_record = {
                        "page": crawled_page.page,
                        "url": crawled_page.final_url or crawled_page.url,
                        "replies": [
                            {
                                "post_id": post.post_id,
                                "author": post.author,
                                "created_at": post.created_at,
                                "text": post.text,
                            }
                            for post in replies
                        ],
                    }
                    # The page must be durable before the checkpoint moves past it.
                    await asyncio.to_thread(
                        self.storage.append_page, paths.pages_path, page_record
                    )
                    saved_pages[crawled_page.page] = page_record
                    posts_written = self._posts_written(saved_pages)
                    await asyncio.to_thread(
                        self.storage.write_checkpoint,
                        paths.checkpoint_path,
                        checkpoint(current_page),
                    )
                else:
                    await asyncio.to_thread(
                        self.storage.write_checkpoint,
                        paths.checkpoint_path,
                        checkpoint(current_page - 1),
                    )
                    if (
                        crawled_page.kind == "blocked"
                        and not retried_with_fresh_profile
                    ):
                        browser_client = self._browser_client(True)
                        retried_with_fresh_profile = True
                        continue
                    stopped_reason = crawled_page.kind
                    break

                if on_progress:
                    on_progress(
                        CrawlProgress(
                            current_page=current_page,
                            last_page=last_page,
                            pages_crawled=len(pages),
                            posts_written=posts_written,
                        )
                    )

                if current_page >= pages_requested:
                    break

                current_page += 1
        finally:
            # Also runs on cancellation or errors so the assembled JSON always
            # matches the durable page log.
            await asyncio.to_thread(
                self.storage.write_crawl, paths.output_path, url, saved_pages
            )

        return ThreadCrawlResult(
            thread_id=thread_id,
            url=url,
            output_path=str(paths.output_path),
            pages_path=str(paths.pages_path),
            checkpoint_path=str(paths.checkpoint_path),
            pages_requested=max(pages_requested, last_page),
            pages_crawled=len(pages),
            posts_written=posts_written,
            last_page=last_page,
            mode=mode,
            resume_from_page=resume_from_page,
            stopped_reason=stopped_reason,
            pages=pages,
        )

    async def _fetch_page(
        self,
        url: str,
        page: int,
        thread_id: str,
        cookie: str | None,
        timeout_seconds: int,
        browser_client: BrowserClient,
        thread_dir: Path,
    ) -> tuple[CrawledPage, list]:
        started = time.perf_counter()
        try:
            result = await browser_client.acquire(
                url,
                timeout_seconds,
                cookie=cookie,
            )
        except BrowserAcquireError as exc:
            return (
                CrawledPage(
                    page=page,
                    url=url,
                    kind="network_error",
                    load_ms=self._elapsed_ms(started),
                    error=str(exc),
                ),
                [],
            )
        load_ms = self._elapsed_ms(started)

        soup = self.parser.parse_html(result.body)
        kind = classify_page(result.status_code, soup)
        posts = self.parser.extract_posts(
            soup,
            page=page,
            page_url=result.final_url or url,
            thread_id=thread_id,
        )
        if 200 <= result.status_code < 300 and posts:
            kind = "ok"
        elif kind == "ok":
            # A real thread page always has posts; an empty one means the
            # page did not render or the thread is gone.
            kind = "no_posts"
        last_page = max(1, self.parser.parse_last_page(soup))

        snapshot_path = ""
        if kind != "ok":
            snapshot = await asyncio.to_thread(
                self.storage.write_snapshot, thread_dir, page, kind, result.body
            )
            snapshot_path = str(snapshot)

        return (
            CrawledPage(
                page=page,
                url=url,
                final_url=result.final_url,
                status_code=result.status_code,
                kind=kind,
                bytes=len(result.body.encode("utf-8")),
                post_count=len(posts),
                last_page=last_page,
                load_ms=load_ms,
                snapshot_path=snapshot_path,
            ),
            posts,
        )

    @staticmethod
    def _elapsed_ms(started: float) -> int:
        return round((time.perf_counter() - started) * 1000)

    @staticmethod
    def _update_start_page(saved_pages: dict[int, dict]) -> int:
        """Fill the first gap, otherwise refresh the last saved page.

        The last saved page may have gained replies since it was crawled.
        """
        first_gap = next(
            page for page in range(1, len(saved_pages) + 2) if page not in saved_pages
        )
        return min(first_gap, max(saved_pages))

    @staticmethod
    def _posts_written(saved_pages: dict[int, dict]) -> int:
        return sum(len(page["replies"]) for page in saved_pages.values())

    def _pages_requested(self, last_page: int, max_pages: int | None) -> int:
        if max_pages is None:
            return last_page
        return min(last_page, max_pages)

    def _browser_client(self, fresh_profile: bool) -> BrowserClient:
        if not fresh_profile:
            return self.browser_client
        if isinstance(self.browser_client, PlaywrightBrowserClient):
            return PlaywrightBrowserClient(
                executable_path=self.browser_client.executable_path,
                headless=self.browser_client.headless,
                user_agent=self.browser_client.user_agent,
                no_sandbox=self.browser_client.no_sandbox,
                persistent_profile=False,
                block_resources=self.browser_client.block_resources,
            )
        return self.browser_client
