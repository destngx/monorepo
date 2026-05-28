from pathlib import Path
from unittest.mock import patch

from fastapi.testclient import TestClient

from src.config import WebCrawlerConfig
from src.main import app


def _bootstrap_config() -> dict[str, str]:
    chrome_bin = Path("/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
    return {
        "CHROME_BIN": str(chrome_bin if chrome_bin.exists() else Path(__file__)),
        "BROWSER_PROFILE_DIR": "./data/browser-profile",
        "CRAWL_OUTPUT_DIR": "./data/crawls",
    }


def test_health_endpoint():
    with patch.multiple("src.main.WebCrawlerConfig", **_bootstrap_config()):
        with TestClient(app) as client:
            response = client.get("/health")
            assert response.status_code == 200
            payload = response.json()
            assert payload["status"] == "ok"
            assert payload["service"] == "web-crawler"


def test_openapi_available():
    with patch.multiple("src.main.WebCrawlerConfig", **_bootstrap_config()):
        with TestClient(app) as client:
            response = client.get("/openapi.json")
            assert response.status_code == 200
            schema = response.json()
            assert schema["info"]["title"] == "Web Crawler"
            assert "/v1/pages/probe/browser" in schema["paths"]


def test_validate_prerequisites():
    with patch.object(WebCrawlerConfig, "CHROME_BIN", __file__):
        WebCrawlerConfig.validate_prerequisites()
