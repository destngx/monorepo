from src.models.browser import BrowserProbeStep
from src.modules.sites.voz.parser import VozParser

from .browser_client import BrowserAcquireError, BrowserClient
from .detector import classify_page


class BrowserProbeService:
    def __init__(
        self,
        browser_client: BrowserClient,
        parser: VozParser | None = None,
        renderer_name: str = "playwright",
    ) -> None:
        self.browser_client = browser_client
        self.parser = parser or VozParser()
        self.renderer_name = renderer_name

    async def probe(
        self,
        url: str,
        page_url: str | None = None,
        cookie: str | None = None,
        timeout_seconds: int = 45,
    ) -> BrowserProbeStep:
        target_url = page_url.strip() if page_url and page_url.strip() else url
        step = BrowserProbeStep(
            name="browser_probe",
            renderer=self.renderer_name,
            url=target_url,
        )

        try:
            result = await self.browser_client.acquire(
                target_url, timeout_seconds, cookie=cookie
            )
        except BrowserAcquireError as exc:
            step.kind = "network_error"
            step.error = str(exc)
            return step

        step.url = result.url
        step.final_url = result.final_url or target_url
        step.status_code = result.status_code
        step.content_type = result.content_type
        step.bytes = len(result.body.encode("utf-8"))

        try:
            soup = self.parser.parse_html(result.body)
            step.kind = classify_page(result.status_code, soup)
            step.last_page = max(1, self.parser.parse_last_page(soup))
            step.post_count = self.parser.count_posts(soup)
            if 200 <= step.status_code < 300 and step.post_count > 0:
                step.kind = "ok"
        except Exception as exc:
            step.kind = step.kind or "parse_error"
            if not step.error:
                step.error = str(exc)

        return step
