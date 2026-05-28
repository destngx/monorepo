import sys
import types

import anyio

from src.services.browser_client import PlaywrightBrowserClient


class FakeResponse:
    status = 200
    headers = {"content-type": "text/html; charset=utf-8"}


class FakePage:
    def __init__(self):
        self.url = "https://voz.vn/t/example.123/"

    async def goto(self, url: str, wait_until: str, timeout: int):
        _ = (url, wait_until, timeout)
        return FakeResponse()

    async def wait_for_timeout(self, milliseconds: int):
        _ = milliseconds

    async def content(self):
        return "<html><body><article id='post-1'>ok</article></body></html>"


class FakeContext:
    def __init__(self):
        self.cookies = []
        self.closed = False
        self.new_page_called = False

    async def add_cookies(self, cookies):
        self.cookies.extend(cookies)

    async def new_page(self):
        self.new_page_called = True
        return FakePage()

    async def close(self):
        self.closed = True


class FakeBrowser:
    def __init__(self):
        self.new_context_kwargs = None
        self.context = FakeContext()
        self.closed = False

    async def new_context(self, **kwargs):
        self.new_context_kwargs = kwargs
        return self.context

    async def close(self):
        self.closed = True


class FakeChromium:
    def __init__(self):
        self.launch_kwargs = None
        self.launch_persistent_kwargs = None
        self.browser = FakeBrowser()
        self.persistent_context = FakeContext()

    async def launch(self, **kwargs):
        self.launch_kwargs = kwargs
        return self.browser

    async def launch_persistent_context(self, **kwargs):
        self.launch_persistent_kwargs = kwargs
        return self.persistent_context


class FakePlaywright:
    def __init__(self):
        self.chromium = FakeChromium()


class AsyncPlaywrightContextManager:
    def __init__(self, playwright: FakePlaywright):
        self.playwright = playwright

    async def __aenter__(self):
        return self.playwright

    async def __aexit__(self, exc_type, exc, tb):
        _ = (exc_type, exc, tb)
        return False


def _install_fake_playwright(monkeypatch, playwright: FakePlaywright):
    playwright_module = types.ModuleType("playwright")
    async_api_module = types.ModuleType("playwright.async_api")
    async_api_module.async_playwright = lambda: AsyncPlaywrightContextManager(
        playwright
    )
    playwright_module.async_api = async_api_module
    monkeypatch.setitem(sys.modules, "playwright", playwright_module)
    monkeypatch.setitem(sys.modules, "playwright.async_api", async_api_module)


def test_acquire_uses_persistent_context_when_enabled(tmp_path, monkeypatch):
    playwright = FakePlaywright()
    _install_fake_playwright(monkeypatch, playwright)

    profile_dir = tmp_path / "profile"
    client = PlaywrightBrowserClient(
        executable_path="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
        headless=True,
        user_agent="test-agent",
        no_sandbox=True,
        persistent_profile=True,
        profile_dir=str(profile_dir),
    )

    result = anyio.run(
        client.acquire,
        "https://voz.vn/t/example.123/",
        10,
        "voz_session=abc123; theme=dark",
    )

    assert result.status_code == 200
    assert result.body.startswith("<html>")
    assert playwright.chromium.launch_persistent_kwargs is not None
    assert playwright.chromium.launch_kwargs is None
    assert playwright.chromium.launch_persistent_kwargs["user_data_dir"] == str(
        profile_dir
    )
    assert playwright.chromium.launch_persistent_kwargs["user_agent"] == "test-agent"
    assert playwright.chromium.launch_persistent_kwargs["args"][-1] == "--no-sandbox"
    assert playwright.chromium.persistent_context.cookies[0]["name"] == "voz_session"
    assert profile_dir.exists()


def test_acquire_uses_ephemeral_context_when_persistent_profile_disabled(
    monkeypatch,
):
    playwright = FakePlaywright()
    _install_fake_playwright(monkeypatch, playwright)

    client = PlaywrightBrowserClient(
        executable_path=None,
        headless=False,
        user_agent="test-agent",
        persistent_profile=False,
    )

    result = anyio.run(
        client.acquire,
        "https://voz.vn/t/example.123/",
        10,
        None,
    )

    assert result.status_code == 200
    assert playwright.chromium.launch_kwargs is not None
    assert playwright.chromium.launch_persistent_kwargs is None
    assert playwright.chromium.browser.new_context_kwargs["user_agent"] == "test-agent"
