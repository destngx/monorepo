import json
from datetime import datetime, timezone

import httpx
import pytest

from src.market.candles import ICT, Timeframe
from src.market.errors import NotFound, Unsupported
from src.market.models import SeriesType, TickerType
from src.sources.algolab import AlgolabSource
from src.sources.http import StatusError
from src.sources.mcp import McpError

pytestmark = pytest.mark.anyio

NOW = datetime(2026, 10, 14, 8, 0, tzinfo=timezone.utc)  # 15:00 in Vietnam


def sse(message: dict) -> httpx.Response:
    """An answer the way Algolab sends it: one event-stream message."""
    return httpx.Response(200, headers={"content-type": "text/event-stream"}, text=f"event: message\ndata: {json.dumps(message)}\n\n")


def tool_answer(request_id, payload, is_error=False) -> httpx.Response:
    result = {"content": [{"type": "text", "text": json.dumps(payload)}]}
    if is_error:
        result["isError"] = True
    return sse({"jsonrpc": "2.0", "id": request_id, "result": result})


class FakeMcp:
    """Records JSON-RPC calls; `tools` maps a tool name to a payload or a function of its arguments."""

    def __init__(self, tools=None, session=None):
        self.calls = []
        self.headers = []
        self.tools = tools or {}
        self.session = session

    def __call__(self, request: httpx.Request) -> httpx.Response:
        message = json.loads(request.content)
        self.calls.append((request.url.path, message.get("method"), message.get("params")))
        self.headers.append(request.headers)
        if message["method"] == "initialize":
            headers = {"mcp-session-id": self.session} if self.session else {}
            response = sse({"jsonrpc": "2.0", "id": message["id"], "result": {"serverInfo": {"name": "algolab-market-data"}}})
            response.headers.update(headers)
            return response
        if message["method"] == "notifications/initialized":
            return httpx.Response(202)
        name, arguments = message["params"]["name"], message["params"]["arguments"]
        answer = self.tools[name]
        answer = answer(arguments) if callable(answer) else answer
        if isinstance(answer, httpx.Response):
            return answer
        return tool_answer(message["id"], answer, is_error=answer.get("status") != "ok")

    def tool_calls(self, name=None):
        return [params for _, method, params in self.calls if method == "tools/call" and (name is None or params["name"] == name)]


def source(fake, api_key=""):
    return AlgolabSource(base_url="https://algolab.test", api_key=api_key, transport=httpx.MockTransport(fake), now=lambda: NOW)


def ok(data, **meta):
    return {"status": "ok", "data": data, "meta": {"quota": {"used": 1, "limit": 1000}, **meta}}


class TestAlgolabConnection:
    async def test_without_a_key_uses_the_free_endpoint(self):
        fake = FakeMcp()
        assert await source(fake).health() == "ok"
        assert [(path, method) for path, method, _ in fake.calls] == [("/free", "initialize"), ("/free", "notifications/initialized")]
        assert "authorization" not in fake.headers[0]

    async def test_a_key_uses_the_account_endpoint(self):
        fake = FakeMcp()
        algolab = source(fake, api_key="alm_secret")
        await algolab.health()
        assert fake.calls[0][0] == "/"
        assert fake.headers[0]["authorization"] == "Bearer alm_secret"
        assert algolab.keyed

    async def test_the_session_is_opened_once_and_its_id_sent(self):
        fake = FakeMcp({"get_quotes": ok({"quotes": [{"symbol": "FPT", "kind": "stock", "price": 57100}]})}, session="s-1")
        algolab = source(fake)
        await algolab.get_ticker("FPT", TickerType.EQUITY)
        await algolab.get_ticker("FPT", TickerType.EQUITY)
        assert [method for _, method, _ in fake.calls].count("initialize") == 1
        assert fake.headers[-1]["mcp-session-id"] == "s-1"

    async def test_an_expired_session_is_reopened_once(self):
        answers = iter([httpx.Response(404), None])

        def quotes(arguments):
            answer = next(answers)
            return answer or ok({"quotes": [{"symbol": "FPT", "kind": "stock", "price": 57100}]})

        fake = FakeMcp({"get_quotes": quotes}, session="s-1")
        ticker = await source(fake).get_ticker("FPT", TickerType.EQUITY)
        assert ticker.price == 57.1
        assert [method for _, method, _ in fake.calls].count("initialize") == 2

    async def test_http_and_json_rpc_errors_raise(self):
        with pytest.raises(StatusError, match="401"):
            await source(FakeMcp({"get_quotes": httpx.Response(401)})).get_ticker("FPT", TickerType.EQUITY)

        def rpc_error(_):  # request 1 was initialize
            return sse({"jsonrpc": "2.0", "id": 2, "error": {"code": -32602, "message": "bad params"}})

        with pytest.raises(McpError, match="bad params"):
            await source(FakeMcp({"get_quotes": rpc_error})).get_ticker("FPT", TickerType.EQUITY)


class TestAlgolabTicker:
    async def test_stock_prices_are_in_thousands_of_vnd_like_vnstock(self):
        fake = FakeMcp({"get_quotes": ok({"quotes": [{"symbol": "FPT", "kind": "stock", "price": 57100, "updated_at": "2026-10-14T07:45:00+00:00"}]})})
        ticker = await source(fake).get_ticker(" fpt ", TickerType.EQUITY)
        assert (ticker.symbol, ticker.price, ticker.currency, ticker.provider) == ("FPT", 57.1, "VND", "algolab")
        assert ticker.timestamp == "2026-10-14T07:45:00+00:00"
        assert fake.tool_calls("get_quotes") == [{"name": "get_quotes", "arguments": {"symbols": ["FPT"]}}]

    async def test_indices_stay_in_points(self):
        fake = FakeMcp({"get_quotes": ok({"quotes": [{"symbol": "VNINDEX", "kind": "index", "price": 1704.87}]})})
        assert (await source(fake).get_ticker("VNINDEX", TickerType.EQUITY)).price == 1704.87

    async def test_a_symbol_missing_from_the_quotes_is_not_found(self):
        fake = FakeMcp({"get_quotes": ok({"quotes": []}, notes=["Không có mã: NOPE"])})
        with pytest.raises(NotFound):
            await source(fake).get_ticker("NOPE", TickerType.EQUITY)

    async def test_an_unknown_kind_fails_rather_than_guessing_units(self):
        fake = FakeMcp({"get_quotes": ok({"quotes": [{"symbol": "XYZ", "kind": "bond", "price": 100000}]})})
        with pytest.raises(ValueError, match="unknown instrument kind"):
            await source(fake).get_ticker("XYZ", TickerType.EQUITY)

    @pytest.mark.parametrize("symbol,kind", [("FPT", TickerType.IFC), ("XAUUSD", TickerType.EQUITY)])
    async def test_other_types_and_global_symbols_are_unsupported(self, symbol, kind):
        fake = FakeMcp()
        with pytest.raises(Unsupported):
            await source(fake).get_ticker(symbol, kind)
        assert fake.calls == []

    async def test_tool_errors_raise_and_unknown_symbols_are_not_found(self):
        missing = {"status": "error", "error": {"code": "SYMBOL_NOT_FOUND", "message": "Không có mã 'NOPE'"}}
        with pytest.raises(NotFound, match="NOPE"):
            await source(FakeMcp({"get_quotes": missing})).get_ticker("NOPE", TickerType.EQUITY)
        quota = {"status": "error", "error": {"code": "QUOTA_EXCEEDED", "message": "hết hạn mức"}}
        with pytest.raises(RuntimeError, match="QUOTA_EXCEEDED"):
            await source(FakeMcp({"get_quotes": quota})).get_ticker("FPT", TickerType.EQUITY)


class TestAlgolabCandles:
    async def test_daily_bars_are_ict_days_in_thousands_of_vnd(self):
        bars = [
            {"t": "2026-10-12", "o": 58100, "h": 58800, "l": 55800, "c": 57100, "v": 100},
            {"t": "2026-10-13", "o": 57000, "h": 57600, "l": 56500, "c": 57300, "v": 200},
        ]
        fake = FakeMcp({"get_price_history": ok({"kind": "stock", "bars": bars}, next_cursor=None)})
        start = datetime(2026, 10, 12, tzinfo=ICT)
        series = await source(fake).get_candles("FPT", Timeframe.D1, start, NOW)
        assert [(c.time, c.open, c.close, c.volume) for c in series.candles] == [
            (datetime(2026, 10, 12, tzinfo=ICT), 58.1, 57.1, 100),
            (datetime(2026, 10, 13, tzinfo=ICT), 57.0, 57.3, 200),
        ]
        assert fake.tool_calls()[0]["arguments"] == {"symbol": "FPT", "interval": "1d", "from_date": "2026-10-12", "to_date": "2026-10-14", "limit": 1000}

    async def test_hourly_bars_keep_their_offset(self):
        bars = [{"t": "2026-10-14T13:00+07:00", "o": 56700, "h": 57200, "l": 56500, "c": 57000, "v": 1}]
        fake = FakeMcp({"get_price_history": ok({"kind": "stock", "bars": bars})})
        series = await source(fake).get_candles("FPT", Timeframe.H1, datetime(2026, 10, 14, tzinfo=ICT), NOW)
        assert series.candles[0].time == datetime(2026, 10, 14, 13, tzinfo=ICT)

    async def test_older_pages_are_followed_until_the_start_is_covered(self):
        pages = {
            None: ok({"kind": "index", "bars": [{"t": "2026-10-13", "c": 1700}]}, next_cursor="older"),
            "older": ok({"kind": "index", "bars": [{"t": "2026-10-01", "c": 1650}]}, next_cursor="oldest"),
        }
        fake = FakeMcp({"get_price_history": lambda arguments: pages[arguments.get("cursor")]})
        series = await source(fake).get_candles("VNINDEX", Timeframe.D1, datetime(2026, 10, 1, tzinfo=ICT), NOW)
        assert [c.close for c in series.candles] == [1650, 1700]
        assert [call["arguments"].get("cursor") for call in fake.tool_calls()] == [None, "older"]

    async def test_paging_stops_after_the_page_limit(self):
        fake = FakeMcp({"get_price_history": ok({"kind": "index", "bars": [{"t": "2026-10-13", "c": 1700}]}, next_cursor="more")})
        await source(fake).get_candles("VNINDEX", Timeframe.D1, datetime(2000, 1, 1, tzinfo=ICT), NOW)
        assert len(fake.tool_calls()) == 4

    async def test_four_hour_bars_are_left_to_the_service(self):
        with pytest.raises(Unsupported):
            await source(FakeMcp()).get_candles("FPT", Timeframe.H4, NOW, NOW)

    async def test_equity_series_uses_the_last_30_days(self):
        fake = FakeMcp({"get_price_history": ok({"kind": "stock", "bars": [{"t": "2026-10-13", "o": 1000, "h": 2000, "l": 500, "c": 1500}]})})
        series = await source(fake).get_price_series("FPT", SeriesType.EQUITY)
        assert series.data == [{"time": "2026-10-13 00:00:00", "open": 1.0, "high": 2.0, "low": 0.5, "close": 1.5}]
        assert fake.tool_calls()[0]["arguments"]["from_date"] == "2026-09-14"
        with pytest.raises(Unsupported):
            await source(FakeMcp()).get_price_series("XAU", SeriesType.GOLD_USD)
