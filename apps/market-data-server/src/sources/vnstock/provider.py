"""
vnstock (in-process): VN equities and indices. Calls share the server's
vnstock rate limiter and quota guard with /api/v1.
"""

import time
from datetime import datetime, timedelta, timezone
from typing import Callable, List

import pandas as pd

from ...market.candles import ICT, Timeframe
from ...market.errors import NotFound, Unsupported
from ...market.models import Candle, CandleSeries, PriceSeries, SeriesType, Ticker, TickerType
from ...market.symbols import is_vn_listing
from ...modules.shared import deps

# vnstock intervals are case-sensitive ("1M" is a month).
INTERVALS = {Timeframe.H1: "1H", Timeframe.D1: "1D", Timeframe.W1: "1W"}


def _history(symbol: str, start: str, end: str, interval: str) -> pd.DataFrame:
    return deps.quote_client(symbol).history(start=start, end=end, interval=interval)


def _bar_time(value) -> datetime:
    """vnstock reports bar times as naive Vietnam local time."""
    at = pd.Timestamp(value).to_pydatetime()
    return at.replace(tzinfo=ICT) if at.tzinfo is None else at


def _number(value) -> float:
    return float(value) if pd.notnull(value) else 0.0


def _candles(frame: pd.DataFrame) -> List[Candle]:
    if frame is None or frame.empty:
        return []
    return [
        Candle(
            time=_bar_time(row["time"]),
            open=_number(row.get("open")),
            high=_number(row.get("high")),
            low=_number(row.get("low")),
            close=_number(row.get("close")),
            volume=_number(row.get("volume")),
        )
        for row in frame.to_dict(orient="records")
    ]


class VnstockSource:
    name = "vnstock"

    def __init__(self, now: Callable[[], datetime] = lambda: datetime.now(timezone.utc)):
        self._now = now

    async def aclose(self) -> None:
        return None

    async def health(self) -> str:
        # A probe would spend quota; report the local rate limiter instead.
        limiter = deps.get_rate_limiter()
        if limiter and limiter.backoff_until and limiter.backoff_until > time.time():
            return "backing off"
        return "ok"

    async def _bars(self, symbol: str, interval: str, start: datetime, end: datetime) -> List[Candle]:
        frame = await deps.run_vnstock(
            _history, symbol, start.astimezone(ICT).strftime("%Y-%m-%d"), end.astimezone(ICT).strftime("%Y-%m-%d"), interval
        )
        return _candles(frame)

    async def get_candles(self, symbol: str, timeframe: Timeframe, start: datetime, end: datetime) -> CandleSeries:
        if not is_vn_listing(symbol):
            raise Unsupported(f"{symbol} is not a VN listing")
        interval = INTERVALS.get(timeframe)
        if interval is None:
            raise Unsupported(f"vnstock does not serve {timeframe.value} candles")
        candles = await self._bars(symbol, interval, start, end)
        return CandleSeries(symbol=symbol, timeframe=timeframe.value, provider=self.name, currency="VND", candles=candles)

    async def get_ticker(self, symbol: str, ticker_type: TickerType) -> Ticker:
        if ticker_type is not TickerType.EQUITY:
            raise Unsupported(f"vnstock does not serve {ticker_type.value} tickers")
        if not is_vn_listing(symbol):
            raise Unsupported(f"{symbol} is not a VN listing")
        now = self._now()
        # Two weeks covers the longest market holiday (Tet).
        bars = [bar for bar in await self._bars(symbol, "1D", now - timedelta(days=14), now) if bar.close > 0]
        if not bars:
            raise NotFound(f"vnstock has no recent price for {symbol}")
        last = max(bars, key=lambda bar: bar.time)
        return Ticker(symbol=symbol, type=ticker_type, provider=self.name, currency="VND", price=last.close, timestamp=last.time.isoformat())

    async def get_price_series(self, symbol: str, series_type: SeriesType) -> PriceSeries:
        """The last 30 days of daily bars, in the row shape /api/v1/stocks/historical returns."""
        if series_type is not SeriesType.EQUITY:
            raise Unsupported(f"vnstock serves only {SeriesType.EQUITY.value} price series")
        if not is_vn_listing(symbol):
            raise Unsupported(f"{symbol} is not a VN listing")
        now = self._now()
        bars = await self._bars(symbol, "1D", now - timedelta(days=30), now)
        points = [
            {"time": bar.time.strftime("%Y-%m-%d %H:%M:%S"), "open": bar.open, "high": bar.high, "low": bar.low, "close": bar.close}
            for bar in bars
        ]
        return PriceSeries(provider=self.name, series_type=series_type, currency="VND", data=points)
