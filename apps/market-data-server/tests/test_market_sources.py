import json
from datetime import datetime, timezone
from unittest.mock import patch

import httpx
import pandas as pd
import pytest

from src.market.candles import Timeframe
from src.market.errors import NotFound, Unsupported
from src.market.models import RateBasis, SeriesType, TickerType
from src.sources.coingecko import CoinGeckoSource
from src.sources.currencyapi import CurrencyApiSource
from src.sources.fmarket import FmarketSource
from src.sources.http import StatusError
from src.sources.vnstock import VnstockSource
from src.sources.yahoo import YahooSource
from src.sources.yahoo.provider import yahoo_symbol

pytestmark = pytest.mark.anyio

NOW = datetime(2026, 10, 12, 20, 0, tzinfo=timezone.utc)  # 03:00 on the 13th in Vietnam


class Upstream:
    """A MockTransport that records requests and answers from a handler."""

    def __init__(self, handler):
        self.requests = []
        self._handler = handler

    def __call__(self, request: httpx.Request) -> httpx.Response:
        self.requests.append(request)
        answer = self._handler(request)
        if isinstance(answer, httpx.Response):
            return answer
        return httpx.Response(200, json=answer)

    @property
    def transport(self):
        return httpx.MockTransport(self)


def body(request: httpx.Request) -> dict:
    return json.loads(request.content or b"{}")


GOLD_ROWS = {
    "data": [
        # Newest first, as fmarket answers: 2026-10-12 and 2026-10-11 00:00 ICT.
        {"price": 4162.96, "bidSjc": 141300000, "askSjc": 144300000, "reportDate": 1791738000000},
        {"price": 4150, "bidSjc": 141000000, "askSjc": 144000000, "reportDate": 1791651600000},
    ]
}


class TestFmarket:
    def source(self, handler):
        upstream = Upstream(handler)
        return FmarketSource(base_url="https://fmarket.test", transport=upstream.transport, now=lambda: NOW), upstream

    async def test_sends_the_web_app_headers(self):
        source, upstream = self.source(lambda r: {"status": 200, "data": {}})
        assert await source.health() == "ok"
        headers = upstream.requests[0].headers
        assert headers["origin"] == "https://fmarket.vn"
        assert headers["referer"] == "https://fmarket.vn/"
        assert headers["f-language"] == "vi"

    async def test_equity_ticker_is_unsupported(self):
        source, _ = self.source(lambda r: {})
        with pytest.raises(Unsupported):
            await source.get_ticker("ACB", TickerType.EQUITY)

    async def test_xau_is_the_world_gold_price_and_sjc_is_bid_ask(self):
        source, upstream = self.source(lambda r: GOLD_ROWS)
        xau = await source.get_ticker("XAU", TickerType.GOLD)
        assert (xau.symbol, xau.currency, xau.price) == ("XAU", "USD", 4162.96)
        assert xau.timestamp == "2026-10-11T17:00:00Z"
        sjc = await source.get_ticker("SJC", TickerType.GOLD)
        assert (sjc.currency, sjc.bid, sjc.ask) == ("VND", 141300000, 144300000)
        assert upstream.requests[0].method == "POST"

    async def test_ifc_ticker_reads_the_product_nav(self):
        source, upstream = self.source(lambda r: {"data": {"code": "DCDS", "nav": 92958.34, "productNavChange": {"updateAt": 1791738000000}}})
        ticker = await source.get_ticker("dcds/x", TickerType.IFC)
        assert (ticker.symbol, ticker.price, ticker.currency) == ("DCDS", 92958.34, "VND")
        assert upstream.requests[0].url.raw_path == b"/home/product/dcds%2Fx"

    async def test_forex_series_drops_zero_rates_and_rejects_other_pairs(self):
        rows = {"data": [
            {"rateSellUsd": 25890, "reportDate": 1764781200000},
            {"rateSellUsd": 0, "reportDate": 1764694800000},
            {"rateSellUsd": 25850, "reportDate": 1764608400000},
        ]}  # fmt: skip
        source, upstream = self.source(lambda r: rows)
        series = await source.get_price_series("USD_VND", SeriesType.FOREX)
        assert upstream.requests[0].url.path == "/res/get-usd-rate-history"
        assert [p.price for p in series.data["points"]] == [25850, 25890]
        assert (series.currency, series.data["basis"]) == ("VND", RateBasis.BANK_SELL)
        with pytest.raises(Unsupported):
            await source.get_price_series("EUR_VND", SeriesType.FOREX)

    async def test_gold_candles_send_compact_ict_dates_and_map_sjc_to_ask(self):
        source, upstream = self.source(lambda r: GOLD_ROWS)
        start = datetime(2026, 10, 1, 20, 0, tzinfo=timezone.utc)
        series = await source.get_candles("SJC", Timeframe.D1, start, NOW)
        sent = body(upstream.requests[0])
        assert (sent["fromDate"], sent["toDate"]) == ("20261002", "20261013")
        assert series.currency == "VND"
        assert [c.close for c in series.candles] == [144000000, 144300000]
        # XAUUSD history is Yahoo's COMEX future; fmarket's close-only spot would switch instrument.
        for symbol in ("XAUUSD", "FPT"):
            with pytest.raises(Unsupported):
                await source.get_candles(symbol, Timeframe.D1, start, NOW)
        with pytest.raises(Unsupported):
            await source.get_candles("SJC", Timeframe.H1, start, NOW)

    async def test_exchange_rate_uses_the_newest_report(self):
        rows = {"data": [{"rateSellUsd": 26100, "reportDate": 1791738000000}, {"rateSellUsd": 26000, "reportDate": 1791651600000}]}
        source, _ = self.source(lambda r: rows)
        rate = await source.get_exchange_rate("usd", "vnd")
        assert (rate.rate, rate.timestamp, rate.basis) == (26100, "2026-10-11T17:00:00Z", RateBasis.BANK_SELL)
        with pytest.raises(Unsupported):
            await source.get_exchange_rate("EUR", "VND")

    async def test_bank_rates_keep_valid_rates_with_the_table_date(self):
        table = {"data": {"updateAt": "10/10/2026", "bankList": [
            {"name": "Vietcombank", "key": "VCB", "value": "4.6"},
            {"name": "", "key": "ACB", "value": "5.1"},
            {"name": "Broken", "key": "X", "value": "n/a"},
        ]}}  # fmt: skip
        source, _ = self.source(lambda r: table)
        rates = await source.get_bank_rates()
        assert [(r.bank, r.rate, r.tenor, r.updated_at) for r in rates] == [
            ("Vietcombank", 4.6, "", "2026-10-10"),
            ("ACB", 5.1, "", "2026-10-10"),
        ]

    async def test_fund_nav_requests_the_full_history(self):
        source, upstream = self.source(lambda r: {"data": [{"nav": 1}]})
        assert await source.get_fund_nav(28) == [{"nav": 1}]
        assert body(upstream.requests[0]) == {"isAllData": 1, "productId": 28, "navPeriod": "navToBeginning"}

    async def test_funds_filter_ignores_asset_types_for_money_market(self):
        source, upstream = self.source(lambda r: {"data": {"rows": []}})
        await source.get_funds(["BOND"], True, 2, 20)
        sent = body(upstream.requests[0])
        assert (sent["isMMFFund"], sent["fundAssetTypes"], sent["page"], sent["pageSize"]) == (True, [], 2, 20)

    async def test_gold_history_keeps_the_summary_block(self):
        source, upstream = self.source(lambda r: {**GOLD_ROWS, "extra": {"goldRingPriceBuy": 142000000}})
        history = await source.get_gold_history(datetime(2026, 10, 1, tzinfo=timezone.utc), NOW)
        assert history == {"rows": GOLD_ROWS["data"], "summary": {"goldRingPriceBuy": 142000000}}
        assert body(upstream.requests[0]) == {"fromDate": "20261001", "toDate": "20261013", "isAllData": False}

    async def test_error_status_raises(self):
        source, _ = self.source(lambda r: httpx.Response(503))
        with pytest.raises(StatusError, match="503"):
            await source.get_bank_rates()

    async def test_unknown_fund_code_is_not_found(self):
        source, upstream = self.source(lambda r: httpx.Response(400, json={"status": 400, "code": 40065, "data": None}))
        with pytest.raises(NotFound, match="NOPE"):
            await source.get_fund("NOPE")
        assert upstream.requests[0].url.path == "/home/product/NOPE"

    async def test_other_fund_errors_keep_the_status(self):
        source, _ = self.source(lambda r: httpx.Response(400, json={"status": 400, "code": 40000}))
        with pytest.raises(StatusError) as raised:
            await source.get_fund("DCDS")
        assert (raised.value.status, raised.value.code) == (400, 40000)

    async def test_error_code_is_only_read_from_json_bodies(self):
        source, _ = self.source(lambda r: httpx.Response(400, text="40065"))
        with pytest.raises(StatusError) as raised:
            await source.get_fund("DCDS")
        assert raised.value.code is None


def chart(timestamps, closes, currency="VND", price=0.0, market_time=0):
    return {
        "chart": {
            "result": [
                {
                    "meta": {"currency": currency, "regularMarketPrice": price, "regularMarketTime": market_time},
                    "timestamp": timestamps,
                    "indicators": {"quote": [{"open": closes, "high": closes, "low": closes, "close": closes, "volume": [1] * len(closes)}]},
                }
            ],
            "error": None,
        }
    }


class TestYahoo:
    def source(self, handler):
        upstream = Upstream(handler)
        return YahooSource(base_url="https://yahoo.test", transport=upstream.transport, now=lambda: NOW), upstream

    def test_symbol_map(self):
        assert yahoo_symbol("FPT") == "FPT.VN"
        assert yahoo_symbol("VNINDEX") == "^VNINDEX.VN"
        assert yahoo_symbol("VIX") == "^VIX"
        assert yahoo_symbol("USDVND") == "USDVND=X"
        assert yahoo_symbol("XAUUSD") == "GC=F"
        assert yahoo_symbol("BTC-USD") == "BTC-USD"
        for symbol in ("VN30", "HNXINDEX", "UPCOMINDEX", "SJC"):
            with pytest.raises(Unsupported):
                yahoo_symbol(symbol)

    async def test_weekly_equity_maps_symbol_and_keeps_null_bars_for_cleaning(self):
        source, upstream = self.source(lambda r: chart([1791738000, 1792342800], [100500, None]))
        series = await source.get_candles("FPT", Timeframe.W1, datetime(2026, 9, 1, tzinfo=timezone.utc), NOW)
        request = upstream.requests[0]
        assert request.url.path == "/v8/finance/chart/FPT.VN"
        assert request.url.params["interval"] == "1wk"
        assert [c.close for c in series.candles] == [100.5, 0.0]

    async def test_chart_error_is_reported(self):
        source, _ = self.source(lambda r: {"chart": {"result": None, "error": {"code": "Not Found", "description": "No data found"}}})
        with pytest.raises(ValueError, match="No data found"):
            await source.get_candles("FPT", Timeframe.D1, NOW, NOW)

    async def test_vn_index_daily_bars_are_built_from_hourly(self):
        # 09:00 and 10:00 ICT on 2026-10-12, then 09:00 ICT on the 13th.
        source, upstream = self.source(lambda r: chart([1791770400, 1791774000, 1791856800], [1700, 1710, 1720]))
        start = datetime(2020, 1, 1, tzinfo=timezone.utc)
        series = await source.get_candles("VNINDEX", Timeframe.D1, start, NOW)
        params = upstream.requests[0].url.params
        assert params["interval"] == "1h"
        # Clamped to the 729 days Yahoo keeps hourly bars for.
        assert int(params["period1"]) == int((NOW.timestamp())) - 729 * 86400
        assert [c.close for c in series.candles] == [1710, 1720]

    async def test_exchange_rate_reads_the_market_quote(self):
        source, upstream = self.source(lambda r: chart([], [], currency="VND", price=26250.5, market_time=1791738000))
        rate = await source.get_exchange_rate("usd", "vnd")
        assert upstream.requests[0].url.path == "/v8/finance/chart/USDVND%3DX" or upstream.requests[0].url.path == "/v8/finance/chart/USDVND=X"
        assert (rate.base, rate.quote, rate.rate, rate.timestamp, rate.basis) == ("USD", "VND", 26250.5, "2026-10-11T17:00:00Z", RateBasis.MARKET)
        with pytest.raises(Unsupported):
            await source.get_exchange_rate("USDT", "VND")

    async def test_forex_series_is_the_daily_close(self):
        source, _ = self.source(lambda r: chart([1791651600, 1791738000], [26200, 26250], currency="VND"))
        series = await source.get_price_series("USD/VND", SeriesType.FOREX)
        assert (series.data["pair"], series.data["basis"]) == ("USD_VND", RateBasis.MARKET)
        assert [p.price for p in series.data["points"]] == [26200, 26250]

    @pytest.mark.parametrize("symbol", ["FPT", "FPT.VN", "E1VFVN30"])
    async def test_vn_listing_prices_are_in_thousands_of_vnd(self, symbol):
        # Yahoo prices .VN listings in VND; every other source uses thousands.
        source, _ = self.source(lambda r: chart([1791738000], [57100], currency="VND"))
        series = await source.get_candles(symbol, Timeframe.D1, datetime(2026, 10, 1, tzinfo=timezone.utc), NOW)
        candle = series.candles[0]
        assert (candle.open, candle.high, candle.low, candle.close, candle.volume) == (57.1, 57.1, 57.1, 57.1, 1)
        assert series.currency == "VND"

    @pytest.mark.parametrize("symbol,price", [("VIX", 18.5), ("XAUUSD", 4180.2), ("BTC-USD", 65000.0)])
    async def test_points_and_foreign_prices_are_not_scaled(self, symbol, price):
        source, _ = self.source(lambda r: chart([1791738000], [price], currency="USD"))
        series = await source.get_candles(symbol, Timeframe.D1, datetime(2026, 10, 1, tzinfo=timezone.utc), NOW)
        assert series.candles[0].close == price

    async def test_vn_index_is_in_points(self):
        source, _ = self.source(lambda r: chart([1791770400], [1704.87]))
        series = await source.get_candles("VNINDEX", Timeframe.H1, datetime(2026, 10, 1, tzinfo=timezone.utc), NOW)
        assert series.candles[0].close == 1704.87

    async def test_crypto_ticker_is_the_usd_pair(self):
        source, upstream = self.source(lambda r: chart([], [], currency="USD", price=65000.5, market_time=1791738000))
        ticker = await source.get_ticker(" btc ", TickerType.CRYPTO)
        assert upstream.requests[0].url.path == "/v8/finance/chart/BTC-USD"
        assert (ticker.symbol, ticker.price, ticker.currency, ticker.provider, ticker.timestamp) == ("BTC", 65000.5, "USD", "yahoo", "2026-10-11T17:00:00Z")

    async def test_unknown_crypto_is_not_found(self):
        source, _ = self.source(lambda r: httpx.Response(404, json={"chart": {"result": None, "error": {"code": "Not Found"}}}))
        with pytest.raises(NotFound):
            await source.get_ticker("NOPECOIN", TickerType.CRYPTO)

    async def test_crypto_without_a_price_fails_and_other_tickers_are_unsupported(self):
        source, _ = self.source(lambda r: chart([], [], currency="USD", price=0))
        with pytest.raises(ValueError, match="no BTC-USD price"):
            await source.get_ticker("BTC", TickerType.CRYPTO)
        for ticker_type in (TickerType.EQUITY, TickerType.IFC, TickerType.GOLD):
            with pytest.raises(Unsupported):
                await source.get_ticker("FPT", ticker_type)
        with pytest.raises(Unsupported):
            await source.get_ticker("BTC-USD", TickerType.CRYPTO)


class TestCoinGecko:
    def source(self, handler, api_key="demo-key"):
        upstream = Upstream(handler)
        return CoinGeckoSource(base_url="https://cg.test", api_key=api_key, transport=upstream.transport), upstream

    async def test_usdt_in_vnd_uses_the_pinned_id_timestamp_and_key(self):
        source, upstream = self.source(lambda r: {"tether": {"vnd": 26400, "last_updated_at": 1791738000}})
        rate = await source.get_exchange_rate("usdt", "VND")
        request = upstream.requests[0]
        assert request.url.params["ids"] == "tether" and request.url.params["vs_currencies"] == "vnd"
        assert request.headers["x-cg-demo-api-key"] == "demo-key"
        assert (rate.base, rate.quote, rate.rate, rate.timestamp, rate.basis) == ("USDT", "VND", 26400, "2026-10-11T17:00:00Z", RateBasis.CRYPTO_MARKET)

    async def test_unpinned_symbol_resolves_the_best_ranked_exact_match_once(self):
        def handler(request):
            if request.url.path == "/search":
                return {"coins": [
                    {"id": "pepe-fake", "symbol": "PEPE", "market_cap_rank": 900},
                    {"id": "pepe", "symbol": "pepe", "market_cap_rank": 30},
                    {"id": "pepecoin", "symbol": "PEPECOIN", "market_cap_rank": 1},
                    {"id": "unranked", "symbol": "PEPE", "market_cap_rank": None},
                ]}  # fmt: skip
            return {"pepe": {"usd": 0.00001}}

        source, upstream = self.source(handler, api_key="")
        for _ in range(2):
            ticker = await source.get_ticker("pepe", TickerType.CRYPTO)
            assert (ticker.symbol, ticker.price, ticker.currency) == ("PEPE", 0.00001, "USD")
        assert [r.url.path for r in upstream.requests].count("/search") == 1
        assert "x-cg-demo-api-key" not in upstream.requests[0].headers

    async def test_missing_price_fails(self):
        source, _ = self.source(lambda r: {"bitcoin": {}})
        with pytest.raises(ValueError, match="no USD price"):
            await source.get_ticker("BTC", TickerType.CRYPTO)

    async def test_non_crypto_and_other_quotes_are_unsupported(self):
        source, _ = self.source(lambda r: {})
        with pytest.raises(Unsupported):
            await source.get_ticker("FPT", TickerType.EQUITY)
        with pytest.raises(Unsupported):
            await source.get_exchange_rate("BTC", "EUR")


class TestCurrencyApi:
    def source(self, handler):
        upstream = Upstream(handler)
        return CurrencyApiSource(base_url="https://fx.test", transport=upstream.transport), upstream

    async def test_daily_snapshot_rate_with_its_date(self):
        source, upstream = self.source(lambda r: {"date": "2026-10-12", "usd": {"vnd": 26300.12}})
        rate = await source.get_exchange_rate("USD", "VND")
        assert upstream.requests[0].url.path == "/v1/currencies/usd.min.json"
        assert (rate.base, rate.quote, rate.rate, rate.timestamp, rate.basis) == ("USD", "VND", 26300.12, "2026-10-12T00:00:00Z", RateBasis.REFERENCE)

    async def test_unknown_target_fails(self):
        source, _ = self.source(lambda r: {"date": "2026-10-12", "usd": {"eur": 0.9}})
        with pytest.raises(ValueError, match="no USD/VND rate"):
            await source.get_exchange_rate("USD", "VND")

    async def test_upstream_status_is_reported(self):
        source, _ = self.source(lambda r: httpx.Response(404))
        with pytest.raises(StatusError, match="404"):
            await source.get_exchange_rate("USD", "VND")


class TestVnstock:
    async def test_hourly_query_requests_the_resolution_and_reads_ict_times(self):
        frame = pd.DataFrame(
            {
                "time": pd.to_datetime(["2026-10-10 09:00:00", "2026-10-10 10:00:00"]),
                "open": [100.5, 100.8], "high": [101, 102], "low": [100, 100.7], "close": [100.8, 101.9], "volume": [1200, 800],
            }
        )  # fmt: skip
        with patch("src.modules.shared.deps.Quote") as quote:
            quote.return_value.history.return_value = frame
            series = await VnstockSource().get_candles(
                "FPT", Timeframe.H1, datetime(2026, 10, 1, 3, tzinfo=timezone.utc), datetime(2026, 10, 12, 3, tzinfo=timezone.utc)
            )
        quote.return_value.history.assert_called_once_with(start="2026-10-01", end="2026-10-12", interval="1H")
        assert (series.provider, series.currency, len(series.candles)) == ("vnstock", "VND", 2)
        first = series.candles[0]
        assert first.time == datetime(2026, 10, 10, 2, tzinfo=timezone.utc)
        assert (first.close, first.volume) == (100.8, 1200)

    async def test_four_hour_and_non_vn_symbols_are_unsupported(self):
        source = VnstockSource()
        with pytest.raises(Unsupported):
            await source.get_candles("FPT", Timeframe.H4, NOW, NOW)
        with pytest.raises(Unsupported):
            await source.get_candles("XAUUSD", Timeframe.D1, NOW, NOW)

    async def test_ticker_is_the_latest_close(self):
        frame = pd.DataFrame({"time": pd.to_datetime(["2026-10-09", "2026-10-12"]), "close": [120.0, 121.5]})
        with patch("src.modules.shared.deps.Quote") as quote:
            quote.return_value.history.return_value = frame
            ticker = await VnstockSource(now=lambda: NOW).get_ticker("FPT", TickerType.EQUITY)
        assert (ticker.price, ticker.currency, ticker.timestamp) == (121.5, "VND", "2026-10-12T00:00:00+07:00")

    async def test_quota_exhaustion_is_an_error_not_an_exit(self):
        with patch("src.modules.shared.deps.Quote") as quote:
            quote.return_value.history.side_effect = SystemExit("quota")
            with pytest.raises(Exception, match="quota"):
                await VnstockSource(now=lambda: NOW).get_ticker("FPT", TickerType.EQUITY)
