"""
Algolab (https://mcp.algolab.vn): VN stocks, ETFs, warrants, indices and
futures through its MCP server. The last fallback for VN tickers, series and
candles. Without ALGOLAB_API_KEY it uses the keyless `/free` endpoint, whose
quota (120/min, 1,000/day) is shared with every anonymous user, so it only
sits at the end of chains. With a key it uses the account's own quota.
"""

import json
import os
from datetime import date, datetime, timedelta, timezone
from typing import Any, Callable, Dict, List, Optional

import httpx

from ...market.candles import ICT, Timeframe, clean_candles
from ...market.errors import NotFound, Unsupported
from ...market.models import Candle, CandleSeries, PriceSeries, SeriesType, Ticker, TickerType
from ...market.symbols import is_vn_listing, normalise_symbol
from ..http import new_client
from ..mcp import McpClient

DEFAULT_BASE_URL = "https://mcp.algolab.vn"
FREE_PATH = "/free"
KEYED_PATH = "/"

INTERVALS = {Timeframe.H1: "1h", Timeframe.D1: "1d", Timeframe.W1: "1w"}
# Bars per get_price_history call (Algolab's maximum).
PAGE_SIZE = 1000
# Each page spends one call of the quota; 4 pages cover ten years of days.
MAX_PAGES = 4

# Algolab prices stocks, ETFs and warrants in VND; the other sources (and
# WM) use thousands of VND. Indices and futures are points on both.
THOUSAND_VND_KINDS = {"stock", "etf", "cw"}
POINT_KINDS = {"index", "futures_continuous", "futures"}


def _scale(kind: Any) -> float:
    if kind in THOUSAND_VND_KINDS:
        return 1000.0
    if kind in POINT_KINDS:
        return 1.0
    # A new kind may use other units; failing lets the chain move on.
    raise ValueError(f"algolab: unknown instrument kind {kind!r}")


def _number(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _bar_time(raw: Any) -> datetime:
    """Intraday bars carry an ICT offset; daily and weekly bars are ICT dates."""
    text = str(raw or "").strip()
    if len(text) == 10:
        return datetime.combine(date.fromisoformat(text), datetime.min.time(), tzinfo=ICT)
    at = datetime.fromisoformat(text.replace("Z", "+00:00"))
    return at if at.tzinfo else at.replace(tzinfo=ICT)


class AlgolabSource:
    name = "algolab"

    def __init__(
        self,
        base_url: Optional[str] = None,
        api_key: Optional[str] = None,
        transport: Optional[httpx.AsyncBaseTransport] = None,
        now: Callable[[], datetime] = lambda: datetime.now(timezone.utc),
    ):
        key = (api_key if api_key is not None else os.getenv("ALGOLAB_API_KEY") or "").strip()
        # The key is only ever sent as a header; it is never logged.
        headers = {"Authorization": f"Bearer {key}"} if key else {}
        self.keyed = bool(key)
        self._client = new_client(base_url or os.getenv("ALGOLAB_BASE_URL") or DEFAULT_BASE_URL, headers, transport)
        self._now = now
        self._mcp = McpClient(self._client, self.name, KEYED_PATH if key else FREE_PATH)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        # `initialize` spends no quota; a tool call would.
        await self._mcp.connect()
        return "ok"

    async def _tool(self, name: str, arguments: Dict[str, Any]) -> Dict[str, Any]:
        """A tool's `{status, data, meta}` answer; errors raise."""
        result = await self._mcp.call_tool(name, arguments)
        texts = [item.get("text") for item in result.get("content") or [] if isinstance(item, dict) and item.get("type") == "text"]
        if not texts:
            raise ValueError(f"algolab: {name} answered without text content")
        try:
            payload = json.loads(texts[0])
        except ValueError as e:
            raise ValueError(f"algolab: unreadable {name} answer: {e}") from e
        if not isinstance(payload, dict):
            raise ValueError(f"algolab: unreadable {name} answer")
        if result.get("isError") or payload.get("status") != "ok":
            error = payload.get("error") or {}
            code, message = error.get("code") or "ERROR", error.get("message") or "tool call failed"
            if code == "SYMBOL_NOT_FOUND":
                raise NotFound(f"algolab: {message}")
            raise RuntimeError(f"algolab: {name}: {code}: {message}")
        if not isinstance(payload.get("data"), dict):
            raise ValueError(f"algolab: {name} answered without data")
        return payload

    def _check_listing(self, symbol: str) -> str:
        symbol = normalise_symbol(symbol)
        if not is_vn_listing(symbol):
            raise Unsupported(f"{symbol} is not a VN listing")
        return symbol

    async def get_ticker(self, symbol: str, ticker_type: TickerType) -> Ticker:
        if ticker_type is not TickerType.EQUITY:
            raise Unsupported(f"algolab does not serve {ticker_type.value} tickers")
        symbol = self._check_listing(symbol)
        payload = await self._tool("get_quotes", {"symbols": [symbol]})
        quotes = payload["data"].get("quotes") or []
        quote = next((q for q in quotes if isinstance(q, dict) and str(q.get("symbol", "")).upper() == symbol), None)
        if quote is None:
            raise NotFound(f"algolab has no quote for {symbol}")
        price = _number(quote.get("price")) / _scale(quote.get("kind"))
        if price <= 0:
            raise NotFound(f"algolab has no price for {symbol}")
        return Ticker(symbol=symbol, type=ticker_type, provider=self.name, currency="VND", price=price, timestamp=quote.get("updated_at"))

    async def _bars(self, symbol: str, interval: str, start: datetime, end: datetime) -> List[Candle]:
        arguments: Dict[str, Any] = {
            "symbol": symbol,
            "interval": interval,
            "from_date": start.astimezone(ICT).strftime("%Y-%m-%d"),
            "to_date": end.astimezone(ICT).strftime("%Y-%m-%d"),
            "limit": PAGE_SIZE,
        }
        candles: List[Candle] = []
        for _ in range(MAX_PAGES):
            payload = await self._tool("get_price_history", arguments)
            scale = _scale(payload["data"].get("kind"))
            bars = payload["data"].get("bars") or []
            for bar in bars:
                if not isinstance(bar, dict):
                    raise ValueError("algolab: unreadable bar")
                candles.append(
                    Candle(
                        time=_bar_time(bar.get("t")),
                        open=_number(bar.get("o")) / scale,
                        high=_number(bar.get("h")) / scale,
                        low=_number(bar.get("l")) / scale,
                        close=_number(bar.get("c")) / scale,
                        volume=_number(bar.get("v")),
                    )
                )
            cursor = (payload.get("meta") or {}).get("next_cursor")
            # Pages run newest to oldest; stop once the range start is covered.
            if not bars or not cursor or min(_bar_time(bar.get("t")) for bar in bars) <= start:
                break
            arguments = {**arguments, "cursor": cursor}
        return [c for c in clean_candles(candles) if start <= c.time <= end]

    async def get_candles(self, symbol: str, timeframe: Timeframe, start: datetime, end: datetime) -> CandleSeries:
        symbol = self._check_listing(symbol)
        interval = INTERVALS.get(timeframe)
        if interval is None:
            raise Unsupported(f"algolab does not serve {timeframe.value} candles")
        candles = await self._bars(symbol, interval, start, end)
        return CandleSeries(symbol=symbol, timeframe=timeframe.value, provider=self.name, currency="VND", candles=candles)

    async def get_price_series(self, symbol: str, series_type: SeriesType) -> PriceSeries:
        """The last 30 days of daily bars, in the row shape vnstock's series uses."""
        if series_type is not SeriesType.EQUITY:
            raise Unsupported(f"algolab serves only {SeriesType.EQUITY.value} price series")
        symbol = self._check_listing(symbol)
        end = self._now()
        bars = await self._bars(symbol, "1d", end - timedelta(days=30), end)
        points = [
            {"time": bar.time.strftime("%Y-%m-%d %H:%M:%S"), "open": bar.open, "high": bar.high, "low": bar.low, "close": bar.close}
            for bar in bars
        ]
        return PriceSeries(provider=self.name, series_type=series_type, currency="VND", data=points)
