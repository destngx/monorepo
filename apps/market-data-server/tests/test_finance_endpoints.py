import pytest
from fastapi.testclient import TestClient
from src.main import app

client = TestClient(app)

def test_balance_sheet():
    response = client.get("/api/v1/finance/balance-sheet?symbol=VND&period=QUARTERLY")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert isinstance(data["data"], list)

def test_income_statement():
    response = client.get("/api/v1/finance/income-statement?symbol=VND&period=QUARTERLY")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert isinstance(data["data"], list)

def test_cash_flow():
    response = client.get("/api/v1/finance/cash-flow?symbol=VND&period=QUARTERLY")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert isinstance(data["data"], list)

def test_ratio():
    response = client.get("/api/v1/finance/ratio?symbol=VND&period=QUARTERLY")
    assert response.status_code == 200
    data = response.json()
    assert data["success"] is True
    assert isinstance(data["data"], list)
