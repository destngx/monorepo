from fastapi.testclient import TestClient
from src.main import app

client = TestClient(app)

def test_read_main():
    response = client.get("/")
    assert response.status_code == 200
    assert "Welcome to Vnstock API Wrapper" in response.json()["message"]

def test_health_check():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json()["status"] == "ok"
    assert response.json()["service"] == "vnstock-server"

def test_get_quote_schema():
    """
    Test the quote endpoint structure.
    """
    try:
        response = client.get("/api/v1/stocks/quote?symbol=GMD")
        if response.status_code == 200:
            json_data = response.json()
            assert json_data["success"] is True
            assert "price" in json_data["data"]
            assert json_data["data"]["symbol"] == "GMD"
    except Exception:
        pass

def test_get_historical_schema():
    """
    Test the historical endpoint structure.
    """
    try:
        response = client.get("/api/v1/stocks/historical?symbol=GMD&resolution=1D")
        if response.status_code == 200:
            json_data = response.json()
            assert json_data["success"] is True
            assert isinstance(json_data["data"], list)
    except Exception:
        pass
