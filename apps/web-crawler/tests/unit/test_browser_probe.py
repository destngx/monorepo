from fastapi.testclient import TestClient
from unittest.mock import patch

from src.main import app
from src.models.browser import BrowserProbeStep
from src.modules.shared.deps import get_browser_probe_service
from src.services.browser_probe import BrowserProbeService


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
            bytes=1024,
            post_count=3,
            last_page=112,
            error="",
        )


def _bootstrap_config() -> dict[str, str]:
    return {
        "CHROME_BIN": __file__,
        "BROWSER_PROFILE_DIR": "./data/browser-profile",
        "CRAWL_OUTPUT_DIR": "./data/crawls",
    }


def test_browser_probe_endpoint():
    app.dependency_overrides[get_browser_probe_service] = (
        lambda: FakeBrowserProbeService()
    )
    try:
        with patch.multiple("src.main.WebCrawlerConfig", **_bootstrap_config()):
            with TestClient(app) as client:
                response = client.post(
                    "/v1/pages/probe/browser",
                    json={
                        "url": "https://voz.vn/t/cao-rau-doi-khi-nhung-thu-tot-nhat-lai-la-nhung-thu-re-tien.913440/",
                        "cookie": "voz_session=abc123; theme=dark",
                        "timeout_seconds": 10,
                    },
                )
                assert response.status_code == 200
                payload = response.json()
                assert payload["step"]["renderer"] == "playwright"
                assert payload["step"]["kind"] == "ok"
                assert payload["step"]["post_count"] == 3
                assert payload["step"]["last_page"] == 112
    finally:
        app.dependency_overrides.clear()


class CookieForwardingBrowserClient:
    def __init__(self):
        self.seen_cookie = None

    async def acquire(self, url: str, timeout_seconds: int, cookie: str | None = None):
        self.seen_cookie = cookie
        return type(
            "Result",
            (),
            {
                "url": url,
                "final_url": url,
                "status_code": 200,
                "body": "<html><article id='post-1'>ok</article></html>",
                "content_type": "text/html",
            },
        )()


def test_browser_probe_forwards_cookie_to_browser_client():
    browser_client = CookieForwardingBrowserClient()
    service = BrowserProbeService(browser_client=browser_client)

    import anyio

    step = anyio.run(
        service.probe,
        "https://voz.vn/t/example.123/",
        None,
        "voz_session=abc123; theme=dark",
        5,
    )

    assert browser_client.seen_cookie == "voz_session=abc123; theme=dark"
    assert step.kind == "ok"
