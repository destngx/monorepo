"""
Canonical market models shared by every source adapter and the /api/v2 API.

Field names match what wealth-management decodes today (MDS-D4), so its client
adapter maps them one to one.
"""

from dataclasses import dataclass, field, fields, is_dataclass
from datetime import datetime, timezone
from enum import Enum
from typing import Any, List, Optional


class TickerType(str, Enum):
    EQUITY = "equity"
    IFC = "ifc"
    GOLD = "gold"
    # A coin priced in USD.
    CRYPTO = "crypto"


class SeriesType(str, Enum):
    EQUITY = "equity"
    GOLD_USD = "gold_usd"
    FOREX = "forex"


class RateBasis(str, Enum):
    """
    What an exchange rate means. Sources in one FX chain quote different
    rates for the same pair, so a fallback changes the basis, not just the
    provider.
    """

    # A bank's published sell rate (what a buyer of USD pays).
    BANK_SELL = "bank_sell"
    # The FX market rate.
    MARKET = "market"
    # A daily reference snapshot.
    REFERENCE = "reference"
    # A coin's price on crypto exchanges.
    CRYPTO_MARKET = "crypto_market"


@dataclass
class Ticker:
    symbol: str
    type: TickerType
    provider: str
    currency: Optional[str] = None
    price: Optional[float] = None
    bid: Optional[float] = None
    ask: Optional[float] = None
    timestamp: Optional[str] = None


@dataclass
class ExchangeRate:
    base: str
    quote: str
    rate: float
    provider: str
    timestamp: Optional[str] = None
    basis: Optional[RateBasis] = None

    def to_json(self) -> dict:
        # "from"/"to" are Python keywords, so the fields are renamed here only.
        basis = self.basis.value if self.basis else None
        return {"from": self.base, "to": self.quote, "rate": self.rate, "provider": self.provider, "timestamp": self.timestamp, "basis": basis}


@dataclass
class PricePoint:
    price: float
    timestamp: str


@dataclass
class PriceSeries:
    provider: str
    series_type: SeriesType
    data: Any
    currency: Optional[str] = None

    def to_json(self) -> dict:
        return {"provider": self.provider, "seriesType": self.series_type.value, "currency": self.currency, "data": to_json(self.data)}


class RateChannel(str, Enum):
    """Where a deposit rate applies; banks often pay more online."""

    ONLINE = "online"
    COUNTER = "counter"


# tenor_months of a no-term (không kỳ hạn) deposit: withdrawable any time.
NO_TERM = 0
# The longest tenor read from a source or accepted in a filter (five years).
MAX_TENOR_MONTHS = 60


def tenor_label(months: int) -> str:
    """The `tenor` text of a rate by tenor: "no-term", "1m", "12m"."""
    return "no-term" if months == NO_TERM else f"{months}m"


@dataclass
class BankRate:
    """
    A savings rate, percent a year. `tenor` is free text, empty when the
    source publishes one headline rate per bank (fmarket). Sources by tenor
    also set `tenor_months` (0 = no-term), `instant_access` (true only for
    no-term: money can leave at any time without losing interest) and
    `channel` when the source says whether the rate is online or at the
    counter.
    """

    bank: str
    rate: float
    tenor: str = ""
    currency: Optional[str] = None
    updated_at: Optional[str] = None
    tenor_months: Optional[int] = None
    channel: Optional[RateChannel] = None
    instant_access: Optional[bool] = None

    @classmethod
    def by_tenor(cls, bank: str, months: int, rate: float, channel: Optional[RateChannel] = None, updated_at: Optional[str] = None) -> "BankRate":
        return cls(
            bank=bank,
            rate=rate,
            tenor=tenor_label(months),
            currency="VND",
            updated_at=updated_at,
            tenor_months=months,
            channel=channel,
            instant_access=months == NO_TERM,
        )

    def to_json(self) -> dict:
        return {
            "bank": self.bank,
            "tenor": self.tenor,
            "rate": self.rate,
            "currency": self.currency,
            "updatedAt": self.updated_at,
            "tenorMonths": self.tenor_months,
            "channel": self.channel.value if self.channel else None,
            "instantAccess": self.instant_access,
        }


@dataclass
class Candle:
    """One OHLCV bar; `time` is the bar's open time, timezone-aware."""

    time: datetime
    close: float
    open: float = 0.0
    high: float = 0.0
    low: float = 0.0
    volume: float = 0.0


@dataclass
class CandleSeries:
    symbol: str
    timeframe: str
    provider: str
    candles: List[Candle] = field(default_factory=list)
    currency: Optional[str] = None


def utc_iso(at: datetime) -> str:
    """RFC 3339 in UTC with a Z suffix, the format WM's Go decoder expects."""
    return at.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def to_json(value: Any) -> Any:
    """Turn models (and containers of them) into JSON-ready values, dropping None fields."""
    if hasattr(value, "to_json") and is_dataclass(value):
        return {k: v for k, v in value.to_json().items() if v is not None}
    if is_dataclass(value):
        out = {}
        for f in fields(value):
            item = getattr(value, f.name)
            if item is not None:
                out[f.name] = to_json(item)
        return out
    if isinstance(value, Enum):
        return value.value
    if isinstance(value, datetime):
        return utc_iso(value)
    if isinstance(value, dict):
        return {k: to_json(v) for k, v in value.items()}
    if isinstance(value, (list, tuple)):
        return [to_json(v) for v in value]
    return value
