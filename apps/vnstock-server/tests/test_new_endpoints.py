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
