"""
Every source that can answer the same request must answer in the same units:
thousands of VND for VN stocks, ETFs and warrants, points for indices. A
fallback must never change the scale of a series (MARKET_PLATFORM.md 3.3).
"""

import json
from datetime import datetime, timezone
from unittest.mock import patch

import httpx
import pandas as pd
import pytest

from src.market.candles import ICT, Timeframe
from src.sources.algolab import AlgolabSource
from src.sources.vnstock import VnstockSource
from src.sources.yahoo import YahooSource

pytestmark = pytest.mark.anyio

NOW = datetime(2026, 10, 14, 8, 0, tzinfo=timezone.utc)
START = datetime(2026, 10, 1, tzinfo=ICT)
DAY = datetime(2026, 10, 13, tzinfo=ICT)

# One daily bar as each upstream publishes it: (unit price in VND, kind).
CASES = [
    pytest.param("FPT", 57100, "stock", 57.1, id="stock"),
    pytest.param("E1VFVN30", 27450, "etf", 27.45, id="etf"),
    pytest.param("VNINDEX", 1704.87, "index", 1704.87, id="index"),
]


async def vnstock_close(symbol, canonical):
    # vnstock already publishes thousands of VND and points.
    frame = pd.DataFrame({"time": pd.to_datetime([DAY.strftime("%Y-%m-%d")]), "open": [canonical], "high": [canonical], "low": [canonical], "close": [canonical], "volume": [1]})
    with patch("src.modules.shared.deps.Quote") as quote:
        quote.return_value.history.return_value = frame
        series = await VnstockSource(now=lambda: NOW).get_candles(symbol, Timeframe.D1, START, NOW)
    return series.candles[-1].close


async def yahoo_close(symbol, raw):
    def handler(request):
        # VN indices are rebuilt from 1h bars; one hourly bar at 10:00 ICT is enough.
        at = int(DAY.replace(hour=10).timestamp())
        quote = {"open": [raw], "high": [raw], "low": [raw], "close": [raw], "volume": [1]}
        return httpx.Response(200, json={"chart": {"result": [{"meta": {"currency": "VND"}, "timestamp": [at], "indicators": {"quote": [quote]}}], "error": None}})

    source = YahooSource(base_url="https://yahoo.test", transport=httpx.MockTransport(handler), now=lambda: NOW)
    series = await source.get_candles(symbol, Timeframe.D1, START, NOW)
    return series.candles[-1].close


async def algolab_close(symbol, raw, kind):
    def handler(request):
        message = json.loads(request.content)
        if message["method"] == "notifications/initialized":
            return httpx.Response(202)
        result = {"serverInfo": {}}
        if message["method"] == "tools/call":
            payload = {"status": "ok", "data": {"kind": kind, "bars": [{"t": DAY.strftime("%Y-%m-%d"), "o": raw, "h": raw, "l": raw, "c": raw, "v": 1}]}, "meta": {}}
            result = {"content": [{"type": "text", "text": json.dumps(payload)}]}
        body = json.dumps({"jsonrpc": "2.0", "id": message["id"], "result": result})
        return httpx.Response(200, headers={"content-type": "text/event-stream"}, text=f"data: {body}\n\n")

    source = AlgolabSource(base_url="https://algolab.test", api_key="", transport=httpx.MockTransport(handler), now=lambda: NOW)
    series = await source.get_candles(symbol, Timeframe.D1, START, NOW)
    return series.candles[-1].close


@pytest.mark.parametrize("symbol,raw,kind,canonical", CASES)
async def test_every_candle_source_answers_in_the_same_units(symbol, raw, kind, canonical):
    closes = {
        "vnstock": await vnstock_close(symbol, canonical),
        "yahoo": await yahoo_close(symbol, raw),
        "algolab": await algolab_close(symbol, raw, kind),
    }
    assert closes == {"vnstock": canonical, "yahoo": canonical, "algolab": canonical}
