from fastapi.testclient import TestClient
from src.main import app
from unittest.mock import patch, MagicMock
import pandas as pd

client = TestClient(app)

@patch("src.modules.shared.deps.Vnstock")
def test_get_all_symbols(mock_vnstock):
    # Mocking the client.stock().listing.all_symbols()
    mock_df = pd.DataFrame([{"symbol": "VND", "name": "VNDirect"}])
    mock_vnstock.return_value.stock.return_value.listing.all_symbols.return_value = mock_df
    
    response = client.get("/api/v1/listing/stocks")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["symbol"] == "VND"

@patch("src.modules.shared.deps.Vnstock")
def test_listing_bonds(mock_vnstock):
    mock_df = pd.DataFrame([{"symbol": "BOND1", "name": "Bond 1"}])
    mock_vnstock.return_value.stock.return_value.listing.all_bonds.return_value = mock_df
    
    response = client.get("/api/v1/listing/bonds")
    assert response.status_code == 200
    assert response.json()["data"][0]["symbol"] == "BOND1"

@patch("src.modules.shared.deps.Vnstock")
def test_listing_industries(mock_vnstock):
    mock_df = pd.DataFrame([{"industry": "Finance"}])
    mock_vnstock.return_value.stock.return_value.listing.industries_icb.return_value = mock_df
    
    response = client.get("/api/v1/listing/industries-icb")
    assert response.status_code == 200
    assert response.json()["data"][0]["industry"] == "Finance"

@patch("src.modules.shared.deps.Vnstock")
def test_stocks_intraday(mock_vnstock):
    mock_df = pd.DataFrame([{"price": 100, "volume": 10}])
    mock_vnstock.return_value.stock.return_value.quote.intraday.return_value = mock_df
    
    response = client.get("/api/v1/stocks/intraday?symbol=VND")
    assert response.status_code == 200
    assert response.json()["data"][0]["price"] == 100

@patch("src.modules.shared.deps.Vnstock")
def test_price_board(mock_vnstock):
    mock_df = pd.DataFrame([{"symbol": "VND", "price": 100}])
    mock_vnstock.return_value.stock.return_value.trading.price_board.return_value = mock_df
    
    response = client.get("/api/v1/stocks/price-board?symbols=VND,SSI")
    assert response.status_code == 200
    assert response.json()["data"][0]["symbol"] == "VND"

@patch("src.modules.shared.deps.Vnstock")
def test_historical_passes_the_resolution_as_interval(mock_vnstock):
    history = mock_vnstock.return_value.stock.return_value.quote.history
    history.return_value = pd.DataFrame([{"time": "2026-10-01 09:00:00", "close": 100}])

    response = client.get("/api/v1/stocks/historical?symbol=VCB&resolution=1h&start_date=2026-10-01")
    assert response.status_code == 200
    assert response.json()["resolution"] == "1H"
    assert history.call_args.kwargs["interval"] == "1H"

def test_historical_rejects_unknown_resolutions():
    response = client.get("/api/v1/stocks/historical?symbol=VCB&resolution=4H")
    assert response.status_code == 400
