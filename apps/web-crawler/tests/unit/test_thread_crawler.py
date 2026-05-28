import json

import anyio

from src.modules.sites.voz.crawler import VozThreadCrawler


class FakeBrowserClient:
    def __init__(self, pages: dict[str, str], status_code: int = 200):
        self.pages = pages
        self.urls = []
        self.status_code = status_code

    async def acquire(self, url: str, timeout_seconds: int, cookie: str | None = None):
        _ = (timeout_seconds, cookie)
        self.urls.append(url)
        html = self.pages[url]
        return type(
            "Result",
            (),
            {
                "url": url,
                "final_url": url,
                "status_code": self.status_code,
                "body": html,
                "content_type": "text/html",
            },
        )()


def _html(page: int, last_page: int, post_id: int) -> str:
    return f"""
    <html>
      <body>
        <input class="js-pageJumpPage" max="{last_page}" />
        <article class="message message--post js-post" data-author="user{page}" data-content="post-{post_id}" id="js-post-{post_id}">
          <h4 class="message-name"><a href="/u/user{page}.{page}/" class="username" data-user-id="{page}">user{page}</a></h4>
          <time class="u-dt" datetime="2024-01-26T11:38:20+0700" data-timestamp="1706243900"></time>
          <ul class="message-attribution-opposite"><li><a href="/t/example.913440/post-{post_id}">#{page}</a></li></ul>
          <article class="message-body"><div class="bbWrapper">post {page}</div></article>
        </article>
      </body>
    </html>
    """


def test_thread_crawler_writes_structured_json(tmp_path):
    base_url = "https://voz.vn/t/example.913440/"
    page_two_url = "https://voz.vn/t/example.913440/page-2"
    browser = FakeBrowserClient(
        {
            base_url: _html(page=1, last_page=2, post_id=101),
            page_two_url: _html(page=2, last_page=2, post_id=102),
        }
    )
    service = VozThreadCrawler(browser_client=browser, output_dir=str(tmp_path))

    result = anyio.run(
        service.crawl_thread,
        base_url,
        None,
        5,
        None,
        0,
    )

    assert result.thread_id == "913440"
    assert result.pages_crawled == 2
    assert result.posts_written == 2
    assert browser.urls == [base_url, page_two_url]

    payload = json.loads(
        (tmp_path / "voz.vn" / "913440" / "913440.json").read_text(encoding="utf-8")
    )

    assert payload == [
        {
            "url": base_url,
            "pages": [
                {
                    "page": 1,
                    "url": base_url,
                    "replies": [
                        {
                            "author": "user1",
                            "created_at": "2024-01-26T11:38:20+0700",
                            "text": "post 1",
                        }
                    ],
                },
                {
                    "page": 2,
                    "url": page_two_url,
                    "replies": [
                        {
                            "author": "user2",
                            "created_at": "2024-01-26T11:38:20+0700",
                            "text": "post 2",
                        }
                    ],
                },
            ],
        }
    ]


def test_thread_crawler_writes_checkpoint_and_resumes(tmp_path):
    base_url = "https://voz.vn/t/example.913440/"
    page_71_url = "https://voz.vn/t/example.913440/page-71"
    browser = FakeBrowserClient(
        {page_71_url: _html(page=71, last_page=112, post_id=171)}
    )
    service = VozThreadCrawler(browser_client=browser, output_dir=str(tmp_path))

    result = anyio.run(
        service.crawl_thread,
        base_url,
        None,
        5,
        71,
        0,
        True,
        71,
    )

    assert result.resume_from_page == 71
    assert result.pages_crawled == 1
    assert result.pages[0].page == 71
    assert browser.urls == [page_71_url]

    checkpoint = json.loads(
        (tmp_path / "voz.vn" / "913440" / "913440.checkpoint.json").read_text(
            encoding="utf-8"
        )
    )
    assert checkpoint["last_successful_page"] == 71
    assert checkpoint["last_page"] == 112

    payload = json.loads(
        (tmp_path / "voz.vn" / "913440" / "913440.json").read_text(encoding="utf-8")
    )
    assert payload[0]["pages"][0]["page"] == 71


def test_thread_crawler_recovers_blocked_page_with_fresh_profile(tmp_path):
    base_url = "https://voz.vn/t/example.913440/"
    page_71_url = "https://voz.vn/t/example.913440/page-71"

    blocked_browser = FakeBrowserClient(
        {page_71_url: _html(page=71, last_page=112, post_id=171)},
        status_code=403,
    )
    fresh_browser = FakeBrowserClient(
        {page_71_url: _html(page=71, last_page=112, post_id=171)},
    )

    class Service(VozThreadCrawler):
        def _browser_client(self, fresh_profile: bool):
            return fresh_browser if fresh_profile else blocked_browser

    service = Service(browser_client=blocked_browser, output_dir=str(tmp_path))

    result = anyio.run(
        service.crawl_thread,
        base_url,
        None,
        5,
        71,
        0,
        False,
        71,
    )

    assert result.resume_from_page == 71
    assert result.pages_crawled == 2
    assert result.stopped_reason == ""
    assert blocked_browser.urls == [page_71_url]
    assert fresh_browser.urls == [page_71_url]

    payload = json.loads(
        (tmp_path / "voz.vn" / "913440" / "913440.json").read_text(encoding="utf-8")
    )
    assert payload[0]["pages"][0]["page"] == 71
