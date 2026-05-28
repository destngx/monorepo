import os
from pathlib import Path


class WebCrawlerConfig:
    DEBUG = os.getenv("DEBUG", "false").lower() == "true"
    HOST = os.getenv("HOST", "0.0.0.0")
    PORT = int(os.getenv("PORT", "8002"))

    CHROME_BIN = os.getenv(
        "CHROME_BIN",
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    )
    BROWSER_HEADLESS = os.getenv("BROWSER_HEADLESS", "false").lower() == "true"
    BROWSER_TIMEOUT_SECONDS = int(os.getenv("BROWSER_TIMEOUT_SECONDS", "45"))
    BROWSER_PERSISTENT_PROFILE = (
        os.getenv("BROWSER_PERSISTENT_PROFILE", "true").lower() == "true"
    )
    BROWSER_PROFILE_DIR = os.getenv("BROWSER_PROFILE_DIR", "./data/browser-profile")
    CRAWL_OUTPUT_DIR = os.getenv("CRAWL_OUTPUT_DIR", "./data/crawls")
    BROWSER_USER_AGENT = os.getenv(
        "BROWSER_USER_AGENT",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
        "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36",
    )
    BROWSER_NO_SANDBOX = os.getenv("BROWSER_NO_SANDBOX", "false").lower() == "true"

    @staticmethod
    def validate() -> None:
        if WebCrawlerConfig.PORT < 1:
            raise ValueError("PORT must be at least 1")
        if WebCrawlerConfig.BROWSER_TIMEOUT_SECONDS < 1:
            raise ValueError("BROWSER_TIMEOUT_SECONDS must be at least 1")
        if not WebCrawlerConfig.CHROME_BIN.strip():
            raise ValueError("CHROME_BIN must not be empty")
        if (
            WebCrawlerConfig.BROWSER_PERSISTENT_PROFILE
            and not WebCrawlerConfig.BROWSER_PROFILE_DIR.strip()
        ):
            raise ValueError(
                "BROWSER_PROFILE_DIR must not be empty when persistent profile is enabled"
            )

    @staticmethod
    def validate_prerequisites() -> None:
        chrome_bin = Path(WebCrawlerConfig.CHROME_BIN).expanduser()
        if not chrome_bin.exists():
            raise FileNotFoundError(
                f"Chrome executable not found: {chrome_bin}"
            )
        if not chrome_bin.is_file():
            raise FileNotFoundError(
                f"Chrome executable is not a file: {chrome_bin}"
            )

        browser_profile_dir = Path(WebCrawlerConfig.BROWSER_PROFILE_DIR).expanduser()
        browser_profile_dir.mkdir(parents=True, exist_ok=True)
        if not browser_profile_dir.exists() or not browser_profile_dir.is_dir():
            raise FileNotFoundError(
                f"Browser profile directory is not available: {browser_profile_dir}"
            )

        crawl_output_dir = Path(WebCrawlerConfig.CRAWL_OUTPUT_DIR).expanduser()
        crawl_output_dir.mkdir(parents=True, exist_ok=True)
        if not crawl_output_dir.exists() or not crawl_output_dir.is_dir():
            raise FileNotFoundError(
                f"Crawl output directory is not available: {crawl_output_dir}"
            )
