import asyncio
from dataclasses import dataclass
from http.cookies import SimpleCookie
from pathlib import Path
from typing import Protocol


@dataclass(slots=True)
class BrowserAcquireResult:
    url: str
    final_url: str
    status_code: int
    body: str
    content_type: str


class BrowserAcquireError(RuntimeError):
    pass


class BrowserClient(Protocol):
    async def acquire(
        self,
        url: str,
        timeout_seconds: int,
        cookie: str | None = None,
    ) -> BrowserAcquireResult:
        raise NotImplementedError


class PlaywrightBrowserClient:
    def __init__(
        self,
        executable_path: str | None,
        headless: bool,
        user_agent: str,
        no_sandbox: bool = False,
        persistent_profile: bool = True,
        profile_dir: str = "./data/browser-profile",
    ) -> None:
        self.executable_path = executable_path or None
        self.headless = headless
        self.user_agent = user_agent
        self.no_sandbox = no_sandbox
        self.persistent_profile = persistent_profile
        self.profile_dir = Path(profile_dir).expanduser()
        self._persistent_profile_lock = (
            asyncio.Lock() if self.persistent_profile else None
        )

    async def acquire(
        self,
        url: str,
        timeout_seconds: int,
        cookie: str | None = None,
    ) -> BrowserAcquireResult:
        timeout_ms = timeout_seconds * 1000

        try:
            from playwright.async_api import async_playwright

            async with async_playwright() as playwright:
                if self.persistent_profile:
                    return await self._acquire_with_persistent_profile(
                        playwright=playwright,
                        url=url,
                        timeout_ms=timeout_ms,
                        cookie=cookie,
                    )
                return await self._acquire_with_ephemeral_context(
                    playwright=playwright,
                    url=url,
                    timeout_ms=timeout_ms,
                    cookie=cookie,
                )
        except Exception as exc:
            raise BrowserAcquireError(str(exc)) from exc

    async def _acquire_with_persistent_profile(
        self,
        playwright,
        url: str,
        timeout_ms: int,
        cookie: str | None = None,
    ) -> BrowserAcquireResult:
        if self._persistent_profile_lock is None:
            raise BrowserAcquireError("persistent profile lock is unavailable")

        self.profile_dir.mkdir(parents=True, exist_ok=True)

        async with self._persistent_profile_lock:
            context = await playwright.chromium.launch_persistent_context(
                user_data_dir=str(self.profile_dir),
                headless=self.headless,
                executable_path=self.executable_path,
                args=self._launch_args(),
                user_agent=self.user_agent,
                ignore_https_errors=True,
                viewport={"width": 1440, "height": 960},
            )
            try:
                return await self._acquire_with_context(
                    context=context,
                    url=url,
                    timeout_ms=timeout_ms,
                    cookie=cookie,
                )
            finally:
                await context.close()

    async def _acquire_with_ephemeral_context(
        self,
        playwright,
        url: str,
        timeout_ms: int,
        cookie: str | None = None,
    ) -> BrowserAcquireResult:
        browser = await playwright.chromium.launch(
            headless=self.headless,
            executable_path=self.executable_path,
            args=self._launch_args(),
        )
        try:
            context = await browser.new_context(
                user_agent=self.user_agent,
                ignore_https_errors=True,
                viewport={"width": 1440, "height": 960},
            )
            try:
                return await self._acquire_with_context(
                    context=context,
                    url=url,
                    timeout_ms=timeout_ms,
                    cookie=cookie,
                )
            finally:
                await context.close()
        finally:
            await browser.close()

    async def _acquire_with_context(
        self,
        context,
        url: str,
        timeout_ms: int,
        cookie: str | None = None,
    ) -> BrowserAcquireResult:
        if cookie and cookie.strip():
            await context.add_cookies(self._cookie_payload(url, cookie))

        page = await context.new_page()
        response = await page.goto(
            url,
            wait_until="domcontentloaded",
            timeout=timeout_ms,
        )
        await self._wait_for_forum_content(page, timeout_ms)
        html = await page.content()
        final_url = page.url
        status_code = response.status if response else 0
        content_type = ""
        if response:
            content_type = response.headers.get("content-type", "")
        return BrowserAcquireResult(
            url=url,
            final_url=final_url,
            status_code=status_code,
            body=html,
            content_type=content_type,
        )

    async def _wait_for_forum_content(self, page, timeout_ms: int) -> None:
        selector = (
            "article.message--post.js-post, "
            "article[data-content^='post-'], "
            ".block--messages"
        )
        selector_timeout_ms = min(timeout_ms, 15_000)

        if not hasattr(page, "wait_for_selector"):
            await page.wait_for_timeout(300)
            return

        try:
            await page.wait_for_selector(selector, timeout=selector_timeout_ms)
        except Exception:
            await page.wait_for_timeout(300)

    def _launch_args(self) -> list[str]:
        args = [
            "--disable-dev-shm-usage",
            "--disable-background-networking",
            "--no-first-run",
            "--no-default-browser-check",
        ]
        if self.no_sandbox:
            args.append("--no-sandbox")
        return args

    def _cookie_payload(self, url: str, raw_cookie: str) -> list[dict[str, str]]:
        parsed = SimpleCookie()
        parsed.load(raw_cookie)

        cookies: list[dict[str, str]] = []
        for name, morsel in parsed.items():
            cookie = {
                "name": name,
                "value": morsel.value,
                "url": url,
            }
            if morsel["path"]:
                cookie["path"] = morsel["path"]
            if morsel["domain"]:
                cookie["domain"] = morsel["domain"]
            cookies.append(cookie)

        return cookies
