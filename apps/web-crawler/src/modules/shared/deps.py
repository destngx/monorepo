from src.config import WebCrawlerConfig
from src.modules.sites.voz.crawler import VozThreadCrawler
from src.modules.sites.voz.parser import VozParser
from src.services.browser_client import PlaywrightBrowserClient
from src.services.browser_probe import BrowserProbeService


class Services:
    def __init__(self) -> None:
        self.config = WebCrawlerConfig
        self.browser_client = PlaywrightBrowserClient(
            executable_path=self.config.CHROME_BIN,
            headless=self.config.BROWSER_HEADLESS,
            user_agent=self.config.BROWSER_USER_AGENT,
            no_sandbox=self.config.BROWSER_NO_SANDBOX,
            persistent_profile=self.config.BROWSER_PERSISTENT_PROFILE,
            profile_dir=self.config.BROWSER_PROFILE_DIR,
        )
        self.browser_probe_service = BrowserProbeService(
            browser_client=self.browser_client,
            parser=VozParser(),
            renderer_name="playwright",
        )
        self.thread_crawler_service = VozThreadCrawler(
            browser_client=self.browser_client,
            parser=VozParser(),
            output_dir=self.config.CRAWL_OUTPUT_DIR,
        )


_services: Services | None = None


def init_services() -> None:
    global _services
    WebCrawlerConfig.validate_prerequisites()
    _services = Services()


def get_services() -> Services:
    global _services
    if _services is None:
        init_services()
    return _services


def get_browser_probe_service() -> BrowserProbeService:
    return get_services().browser_probe_service


def get_thread_crawler_service() -> VozThreadCrawler:
    return get_services().thread_crawler_service


def get_config() -> WebCrawlerConfig:
    return get_services().config
