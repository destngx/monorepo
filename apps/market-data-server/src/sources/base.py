"""
Source adapter contracts. An adapter implements only the capabilities its
upstream serves; the router checks for each method per call and skips
adapters without it. Adapters only translate: no caching, no fallback.
"""

from datetime import datetime
from typing import Any, List, Protocol, runtime_checkable

from ..market.models import BankRate, CandleSeries, ExchangeRate, PriceSeries, SeriesType, Ticker, TickerType


@runtime_checkable
class Source(Protocol):
    name: str

    async def health(self) -> str:
        """Return a status ("ok", "degraded", ...) or raise when the upstream is down."""
        ...

    async def aclose(self) -> None: ...


class TickerSource(Source, Protocol):
    async def get_ticker(self, symbol: str, ticker_type: TickerType) -> Ticker: ...


class ExchangeRateSource(Source, Protocol):
    async def get_exchange_rate(self, base: str, quote: str) -> ExchangeRate: ...


class PriceSeriesSource(Source, Protocol):
    async def get_price_series(self, symbol: str, series_type: SeriesType) -> PriceSeries: ...


class BankRateSource(Source, Protocol):
    async def get_bank_rates(self) -> List[BankRate]: ...


class CandleSource(Source, Protocol):
    async def get_candles(self, symbol: str, timeframe: Any, start: datetime, end: datetime) -> CandleSeries:
        """Bars of one source timeframe (1h, 1d or 1w) between start and end inclusive."""
        ...


# Capability name -> adapter method, used by the router and /api/v2/providers.
CAPABILITIES = {
    "ticker": "get_ticker",
    "fx": "get_exchange_rate",
    "series": "get_price_series",
    "bank_rates": "get_bank_rates",
    "candles": "get_candles",
    "funds": "get_funds",
}


def capabilities_of(source: Source) -> List[str]:
    return [name for name, method in CAPABILITIES.items() if callable(getattr(source, method, None))]
