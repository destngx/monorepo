"""
Yahoo Finance chart API: the candle fallback for vnstock, global symbols
(VIX, gold futures) and fiat exchange rates.
"""

import os
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from typing import Callable, List, Optional
from urllib.parse import quote as url_quote

import httpx

from ...market.candles import Timeframe, aggregate_candles, clean_candles
from ...market.errors import Unsupported
from ...market.models import Candle, CandleSeries, ExchangeRate, PricePoint, PriceSeries, SeriesType, utc_iso
from ...market.symbols import SJC, USDVND, VIX, VN30, VNINDEX, XAUUSD
from ..http import new_client, request_json

DEFAULT_BASE_URL = "https://query1.finance.yahoo.com"

# Yahoo rejects requests without a browser-like user agent.
HEADERS = {"User-Agent": "Mozilla/5.0 (compatible; market-data-server/1.0)"}

INTERVALS = {Timeframe.H1: "1h", Timeframe.D1: "1d", Timeframe.W1: "1wk"}

# Non-VN instruments the market pulse reads.
GLOBAL_SYMBOLS = {VIX: "^VIX", USDVND: "USDVND=X", XAUUSD: "GC=F"}

UNSUPPORTED = {VN30, "HNXINDEX", "UPCOMINDEX", SJC}

# How far back Yahoo serves 1h bars.
MAX_HOURLY_HISTORY = timedelta(days=729)


def yahoo_symbol(symbol: str) -> str:
    """
    Map a canonical symbol to Yahoo: FPT -> FPT.VN, VNINDEX -> ^VNINDEX.VN.
    Symbols that already carry a Yahoo suffix or pair (BTC-USD) pass through.
    """
    if symbol in GLOBAL_SYMBOLS:
        return GLOBAL_SYMBOLS[symbol]
    if symbol == VNINDEX:
        return "^VNINDEX.VN"
    if symbol in UNSUPPORTED:
        raise Unsupported(f"yahoo has no {symbol} series")
    if any(ch in symbol for ch in ".-=^"):
        return symbol
    return symbol + ".VN"


def _is_vn_index(remote: str) -> bool:
    return remote.startswith("^") and remote.endswith(".VN")


@dataclass
class Chart:
    currency: Optional[str]
    market_price: float
    market_time: datetime
    candles: List[Candle]


def _at(values: Optional[list], index: int) -> float:
    # Yahoo sends null for bars without trades; clean_candles drops them.
    if not values or index >= len(values) or values[index] is None:
        return 0.0
    return float(values[index])


class YahooSource:
    name = "yahoo"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None, now: Callable[[], datetime] = lambda: datetime.now(timezone.utc)):
        self._client = new_client(base_url or os.getenv("YAHOO_FINANCE_BASE_URL") or DEFAULT_BASE_URL, HEADERS, transport)
        self._now = now

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        now = self._now()
        await self.get_candles(VNINDEX, Timeframe.D1, now - timedelta(days=7), now)
        return "ok"

    async def _chart(self, remote: str, interval: str, start: datetime, end: datetime) -> Chart:
        payload = await request_json(
            self._client,
            self.name,
            "GET",
            "/v8/finance/chart/" + url_quote(remote, safe=""),
            params={"interval": interval, "period1": int(start.timestamp()), "period2": int(end.timestamp())},
        )
        chart = (payload or {}).get("chart") or {}
        if chart.get("error"):
            error = chart["error"]
            raise ValueError(f"yahoo {error.get('code')}: {error.get('description')}")
        results = chart.get("result") or []
        quotes = (results[0].get("indicators") or {}).get("quote") if results else None
        if not quotes:
            raise ValueError(f"yahoo returned no chart for {remote}")
        result, quote = results[0], quotes[0]
        candles = [
            Candle(
                time=datetime.fromtimestamp(at, tz=timezone.utc),
                open=_at(quote.get("open"), i),
                high=_at(quote.get("high"), i),
                low=_at(quote.get("low"), i),
                close=_at(quote.get("close"), i),
                volume=_at(quote.get("volume"), i),
            )
            for i, at in enumerate(result.get("timestamp") or [])
        ]
        meta = result.get("meta") or {}
        return Chart(
            currency=meta.get("currency"),
            market_price=float(meta.get("regularMarketPrice") or 0),
            market_time=datetime.fromtimestamp(meta.get("regularMarketTime") or 0, tz=timezone.utc),
            candles=candles,
        )

    async def get_candles(self, symbol: str, timeframe: Timeframe, start: datetime, end: datetime) -> CandleSeries:
        interval = INTERVALS.get(timeframe)
        if interval is None:
            raise Unsupported(f"yahoo does not serve {timeframe.value} candles")
        remote = yahoo_symbol(symbol)
        # Yahoo keeps only the latest daily and weekly bar for VN indices but a
        # full hourly history, so those bars are built from 1h data.
        aggregate = _is_vn_index(remote) and timeframe is not Timeframe.H1
        if aggregate:
            interval = INTERVALS[Timeframe.H1]
            start = max(start, end - MAX_HOURLY_HISTORY)
        chart = await self._chart(remote, interval, start, end)
        candles = chart.candles
        if aggregate:
            candles = aggregate_candles(clean_candles(candles), timeframe)
        return CandleSeries(symbol=symbol, timeframe=timeframe.value, provider=self.name, currency=chart.currency, candles=candles)

    async def get_exchange_rate(self, base: str, quote: str) -> ExchangeRate:
        """A fiat pair from the FX market, e.g. USDVND=X."""
        base, quote = base.strip().upper(), quote.strip().upper()
        if len(base) != 3 or len(quote) != 3:
            raise Unsupported("yahoo quotes only ISO currency pairs")
        now = self._now()
        chart = await self._chart(f"{base}{quote}=X", "1d", now - timedelta(days=7), now)
        if chart.market_price <= 0:
            raise ValueError(f"yahoo has no {base}/{quote} quote")
        return ExchangeRate(base=base, quote=quote, rate=chart.market_price, provider=self.name, timestamp=utc_iso(chart.market_time))

    async def get_price_series(self, symbol: str, series_type: SeriesType) -> PriceSeries:
        """The daily close of a fiat pair over 30 days; symbol is a pair such as USD_VND."""
        if series_type is not SeriesType.FOREX:
            raise Unsupported(f"yahoo serves only {SeriesType.FOREX.value} price series")
        pair = symbol.strip().upper().replace("/", "").replace("_", "")
        if len(pair) != 6:
            raise Unsupported("forex symbol must be a pair such as USD_VND")
        now = self._now()
        chart = await self._chart(pair + "=X", "1d", now - timedelta(days=30), now)
        points = [PricePoint(price=c.close, timestamp=utc_iso(c.time)) for c in clean_candles(chart.candles)]
        if not points:
            raise ValueError(f"yahoo returned an empty {pair} series")
        return PriceSeries(provider=self.name, series_type=series_type, currency=chart.currency, data={"pair": f"{pair[:3]}_{pair[3:]}", "points": points})
