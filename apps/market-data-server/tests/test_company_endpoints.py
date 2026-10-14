from fastapi.testclient import TestClient
from src.main import app
from src.modules.shared import deps
from unittest.mock import patch
import pandas as pd

client = TestClient(app)


@patch("src.modules.shared.deps.Company")
def test_company_overview(mock_company):
    mock_company.return_value.overview.return_value = pd.DataFrame(
        [{"symbol": "VND", "overview": "VNDirect Securities", "employees": float("nan")}]
    )

    response = client.get("/api/v1/company/overview?symbol=vnd")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["symbol"] == "VND"
    assert response.json()["data"][0]["employees"] is None
    assert mock_company.call_args.kwargs["symbol"] == "VND"


@patch("src.modules.shared.deps.Company")
def test_company_news_is_trimmed_to_page_size(mock_company):
    mock_company.return_value.news.return_value = pd.DataFrame(
        [{"title": f"News {i}"} for i in range(5)]
    )

    response = client.get("/api/v1/company/news?symbol=VND&page_size=2")
    assert response.status_code == 200
    assert [n["title"] for n in response.json()["data"]] == ["News 0", "News 1"]


@patch("src.modules.shared.deps.Company")
def test_provider_specific_methods_pin_their_source(mock_company):
    mock_company.return_value.insider_trading.return_value = pd.DataFrame([{"name": "A"}])
    mock_company.return_value.ratio_summary.return_value = pd.DataFrame([{"pe": 10}])

    assert client.get("/api/v1/company/insider-deals?symbol=VND").status_code == 200
    assert mock_company.call_args.kwargs["source"] == "KBS"
    assert client.get("/api/v1/company/ratio-summary?symbol=VND").status_code == 200
    assert mock_company.call_args.kwargs["source"] == "VCI"


@patch("src.modules.shared.deps.Company")
def test_upstream_quota_exit_becomes_429_and_backs_off(mock_company):
    # vnai raises SystemExit when its own quota is exhausted.
    mock_company.return_value.overview.side_effect = SystemExit("Rate limit exceeded.\nmore")

    response = client.get("/api/v1/company/overview?symbol=VND")
    assert response.status_code == 429
    assert deps.get_rate_limiter().backoff_until is not None
