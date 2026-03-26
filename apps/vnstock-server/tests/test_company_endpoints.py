from fastapi.testclient import TestClient
from src.main import app
from unittest.mock import patch
import pandas as pd
import pytest

client = TestClient(app)


@patch("src.modules.shared.deps.Vnstock")
def test_company_overview(mock_vnstock):
    mock_df = pd.DataFrame([{"symbol": "VND", "overview": "VNDirect Securities"}])
    mock_vnstock.return_value.stock.return_value.company.overview.return_value = mock_df
    
    response = client.get("/api/v1/company/overview?symbol=VND")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["symbol"] == "VND"


@patch("src.modules.shared.deps.Vnstock")
def test_company_profile(mock_vnstock):
    mock_df = pd.DataFrame([{"symbol": "VND", "profile": "Financial services"}])
    mock_vnstock.return_value.stock.return_value.company.profile.return_value = mock_df
    
    response = client.get("/api/v1/company/profile?symbol=VND")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["symbol"] == "VND"


@patch("src.modules.shared.deps.Vnstock")
def test_company_news(mock_vnstock):
    mock_df = pd.DataFrame([{"title": "News Title", "time": "2026-03-26"}])
    mock_vnstock.return_value.stock.return_value.company.news.return_value = mock_df
    
    response = client.get("/api/v1/company/news?symbol=VND")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["title"] == "News Title"


@patch("src.modules.shared.deps.Vnstock")
def test_finance_get_all(mock_vnstock):
    # Mocking a callable get_all
    mock_df = pd.DataFrame([{"period": "2026-Q1", "assets": 1000}])
    mock_stock = mock_vnstock.return_value.stock.return_value
    mock_stock.finance.get_all.return_value = mock_df
    
    response = client.get("/api/v1/finance/get-all?symbol=VND")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["assets"] == 1000


@patch("src.modules.shared.deps.Vnstock")
def test_finance_get_all_not_callable(mock_vnstock):
    # Mocking s.finance.get_all as a boolean (not callable)
    mock_stock = mock_vnstock.return_value.stock.return_value
    mock_stock.finance.get_all = True
    
    response = client.get("/api/v1/finance/get-all?symbol=VND")
    assert response.status_code == 501
    assert "not callable" in response.json()["detail"]
