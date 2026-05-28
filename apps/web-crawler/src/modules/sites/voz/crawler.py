import asyncio

from src.models.crawl import CrawledPage, CrawlCheckpoint, ThreadCrawlResult
from src.services.browser_client import (
    BrowserAcquireError,
    BrowserClient,
    PlaywrightBrowserClient,
)
from src.modules.shared.crawl_storage import CrawlStorage
from src.services.detector import classify_html
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

    async def crawl_thread(
        self,
        url: str,
        cookie: str | None = None,
        timeout_seconds: int = 45,
        max_pages: int | None = None,
        delay_seconds: float = 1.5,
        fresh_profile: bool = False,
        resume_from_page: int | None = None,
    ) -> ThreadCrawlResult:
        browser_client = self._browser_client(fresh_profile)
        thread_id = self.parser.thread_id_from_url(url)
        paths = self.storage.resolve_thread_paths(url, thread_id)
        output_path = paths.output_path
        checkpoint_path = paths.checkpoint_path
        self.storage.ensure_thread_dir(output_path.parent)
        if resume_from_page is None:
            self.storage.clear_thread_state(output_path, checkpoint_path)
            crawled_data = [{"url": url, "pages": []}]
            resume_from_page = 1
        else:
            crawled_data = self.storage.load_crawl(output_path) or [{"url": url, "pages": []}]
            resume_from_page = max(1, resume_from_page)

        start_page = resume_from_page
        last_page = 1
        pages: list[CrawledPage] = []
        posts_written = sum(
            len(page.get("replies", []))
            for thread in crawled_data
            for page in thread.get("pages", [])
        )
        pages_requested = max(1, resume_from_page)
        stopped_reason = ""
        retried_with_fresh_profile = fresh_profile

        current_page = resume_from_page

        while True:
            if delay_seconds > 0 and current_page != start_page:
                await asyncio.sleep(self.pacing.delay_seconds(delay_seconds))

            current_url = (
                url if current_page == 1 else self.parser.page_url(url, current_page)
            )
            crawled_page, replies = await self._fetch_page(
                url=current_url,
                page=current_page,
                thread_id=thread_id,
                cookie=cookie,
                timeout_seconds=timeout_seconds,
                browser_client=browser_client,
            )
            pages.append(crawled_page)
            last_page = max(last_page, crawled_page.last_page)
            pages_requested = self._pages_requested(last_page, max_pages)
            posts_written += crawled_page.post_count

            if crawled_page.kind == "ok":
                crawled_data[0]["pages"].append(
                    {
                        "page": crawled_page.page,
                        "url": crawled_page.final_url or crawled_page.url,
                        "replies": [
                            {
                                "author": post.author,
                                "created_at": post.created_at,
                                "text": post.text,
                            }
                            for post in replies
                        ],
                    }
                )
                self.storage.write_checkpoint(
                    checkpoint_path,
                    checkpoint=CrawlCheckpoint.for_page(
                        thread_id=thread_id,
                        url=url,
                        last_successful_page=current_page,
                        last_page=last_page,
                        pages_requested=max(pages_requested, last_page),
                        posts_written=posts_written,
                        output_path=str(output_path),
                    ),
                )
            else:
                if crawled_page.kind == "blocked" and not retried_with_fresh_profile:
                    self.storage.write_checkpoint(
                        checkpoint_path,
                        checkpoint=CrawlCheckpoint.for_page(
                            thread_id=thread_id,
                            url=url,
                            last_successful_page=current_page - 1,
                            last_page=last_page,
                            pages_requested=max(pages_requested, last_page),
                            posts_written=posts_written,
                            output_path=str(output_path),
                        ),
                    )
                    browser_client = self._browser_client(True)
                    retried_with_fresh_profile = True
                    continue
                stopped_reason = crawled_page.kind
                self.storage.write_checkpoint(
                    checkpoint_path,
                    checkpoint=CrawlCheckpoint.for_page(
                        thread_id=thread_id,
                        url=url,
                        last_successful_page=current_page - 1,
                        last_page=last_page,
                        pages_requested=max(pages_requested, last_page),
                        posts_written=posts_written,
                        output_path=str(output_path),
                    ),
                )
                break

            if current_page >= pages_requested:
                break

            current_page += 1

        self.storage.write_crawl(output_path, crawled_data)

        return ThreadCrawlResult(
            thread_id=thread_id,
            url=url,
            output_path=str(output_path),
            checkpoint_path=str(checkpoint_path),
            pages_requested=max(pages_requested, last_page),
            pages_crawled=len(pages),
            posts_written=posts_written,
            last_page=last_page,
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
    ) -> tuple[CrawledPage, list]:
        try:
            result = await browser_client.acquire(
                url,
                timeout_seconds,
                cookie=cookie,
            )
        except BrowserAcquireError as exc:
            return (
                CrawledPage(page=page, url=url, kind="network_error", error=str(exc)),
                [],
            )

        kind = classify_html(result.status_code, result.body)
        posts = self.parser.extract_posts(
            result.body,
            page=page,
            page_url=result.final_url or url,
            thread_id=thread_id,
        )
        if 200 <= result.status_code < 300 and posts:
            kind = "ok"
        last_page = max(1, self.parser.parse_last_page(result.body))

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
            ),
            posts,
        )

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
            )
        return self.browser_client
