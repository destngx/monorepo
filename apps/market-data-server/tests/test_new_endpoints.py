from fastapi.testclient import TestClient
from src.main import app
from src.modules.shared import deps
from unittest.mock import patch
import pandas as pd

client = TestClient(app)


@patch("src.modules.shared.deps.Listing")
def test_get_all_symbols(mock_listing):
    mock_listing.return_value.all_symbols.return_value = pd.DataFrame([{"symbol": "VND", "name": "VNDirect"}])

    response = client.get("/api/v1/listing/stocks")
    assert response.status_code == 200
    assert response.json()["success"] is True
    assert response.json()["data"][0]["symbol"] == "VND"


@patch("src.modules.shared.deps.Listing")
def test_listing_series_results_become_records(mock_listing):
    mock_listing.return_value.all_bonds.return_value = pd.Series(["BOND1", "BOND2"], name="symbol")

    response = client.get("/api/v1/listing/bonds")
    assert response.status_code == 200
    assert response.json()["data"] == [{"symbol": "BOND1"}, {"symbol": "BOND2"}]


@patch("src.modules.shared.deps.Listing")
def test_listing_industries_uses_vci(mock_listing):
    mock_listing.return_value.industries_icb.return_value = pd.DataFrame([{"icb_name": "Finance"}])

    response = client.get("/api/v1/listing/industries-icb")
    assert response.status_code == 200
    assert response.json()["data"][0]["icb_name"] == "Finance"
    assert mock_listing.call_args.kwargs["source"] == "VCI"


@patch("src.modules.shared.deps.Listing")
def test_symbols_by_exchange_filters_and_maps_hose(mock_listing):
    mock_listing.return_value.symbols_by_exchange.return_value = pd.DataFrame(
        [{"symbol": "FPT", "exchange": "HSX"}, {"symbol": "SHS", "exchange": "HNX"}]
    )

    response = client.get("/api/v1/listing/symbols-by-exchange?exchange=hose")
    assert response.status_code == 200
    assert [r["symbol"] for r in response.json()["data"]] == ["FPT"]


@patch("src.modules.shared.deps.Listing")
def test_symbols_by_industries_matches_code_or_name(mock_listing):
    mock_listing.return_value.symbols_by_industries.return_value = pd.DataFrame(
        [
            {"symbol": "VCB", "icb_code": "8300", "icb_name": "Ngân hàng"},
            {"symbol": "FPT", "icb_code": "9500", "icb_name": "Công nghệ"},
        ]
    )

    by_code = client.get("/api/v1/listing/symbols-by-industries?industry=8300").json()
    by_name = client.get("/api/v1/listing/symbols-by-industries?industry=công nghệ").json()
    assert [r["symbol"] for r in by_code["data"]] == ["VCB"]
    assert [r["symbol"] for r in by_name["data"]] == ["FPT"]


@patch("src.modules.shared.deps.Quote")
def test_stocks_intraday(mock_quote):
    mock_quote.return_value.intraday.return_value = pd.DataFrame([{"price": 100, "volume": 10}])

    response = client.get("/api/v1/stocks/intraday?symbol=VND")
    assert response.status_code == 200
    assert response.json()["data"][0]["price"] == 100


@patch("src.modules.shared.deps.Trading")
def test_price_board_flattens_multiindex_columns(mock_trading):
    columns = pd.MultiIndex.from_tuples([("listing", "symbol"), ("match", "match_price")])
    mock_trading.return_value.price_board.return_value = pd.DataFrame([["VND", 100]], columns=columns)

    response = client.get("/api/v1/stocks/price-board?symbols=VND,SSI")
    assert response.status_code == 200
    assert response.json()["data"][0] == {"listing_symbol": "VND", "match_match_price": 100}
    assert mock_trading.return_value.price_board.call_args.kwargs["symbols_list"] == ["VND", "SSI"]


@patch("src.modules.shared.deps.Quote")
def test_quote_derives_change_from_previous_close(mock_quote):
    mock_quote.return_value.history.return_value = pd.DataFrame(
        [
            {"time": pd.Timestamp("2026-10-13"), "close": 100.0},
            {"time": pd.Timestamp("2026-10-14"), "close": 102.5},
        ]
    )

    data = client.get("/api/v1/stocks/quote?symbol=FPT").json()["data"]
    assert data["price"] == 102.5
    assert data["change"] == 2.5
    assert data["time"] == "2026-10-14 00:00:00"


@patch("src.modules.shared.deps.Quote")
def test_historical_passes_the_resolution_as_interval(mock_quote):
    history = mock_quote.return_value.history
    history.return_value = pd.DataFrame([{"time": "2026-10-01 09:00:00", "close": 100}])

    response = client.get("/api/v1/stocks/historical?symbol=VCB&resolution=1h&start_date=2026-10-01")
    assert response.status_code == 200
    assert response.json()["resolution"] == "1H"
    assert history.call_args.kwargs["interval"] == "1H"


def test_historical_rejects_unknown_resolutions():
    response = client.get("/api/v1/stocks/historical?symbol=VCB&resolution=4H")
    assert response.status_code == 400


@patch("src.modules.shared.deps.Finance")
def test_finance_accepts_legacy_period_names(mock_finance):
    mock_finance.return_value.balance_sheet.return_value = pd.DataFrame([{"item": "Assets"}])

    response = client.get("/api/v1/finance/balance-sheet?symbol=VND&period=QUARTERLY")
    assert response.status_code == 200
    assert response.json()["period"] == "quarter"
    assert mock_finance.return_value.balance_sheet.call_args.kwargs["period"] == "quarter"

    response = client.get("/api/v1/finance/balance-sheet?symbol=VND&period=ANNUAL")
    assert response.json()["period"] == "year"


def test_finance_rejects_unknown_periods():
    assert client.get("/api/v1/finance/ratio?symbol=VND&period=monthly").status_code == 400


@patch("src.modules.shared.deps.Finance")
def test_finance_get_all_combines_statements_and_costs_three_calls(mock_finance):
    finance = mock_finance.return_value
    finance.balance_sheet.return_value = pd.DataFrame([{"item": "bs"}])
    finance.income_statement.return_value = pd.DataFrame([{"item": "is"}])
    finance.cash_flow.return_value = pd.DataFrame([{"item": "cf"}])

    response = client.get("/api/v1/finance/get-all?symbol=VND&period=year")
    assert response.status_code == 200
    data = response.json()["data"]
    assert data["balance_sheet"][0]["item"] == "bs"
    assert data["income_statement"][0]["item"] == "is"
    assert data["cash_flow"][0]["item"] == "cf"
    assert len(deps.get_rate_limiter().requests_minute) == 3
