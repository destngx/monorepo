"""
Capability services behind /api/v2: validate input, route over the chain,
cache the answer, keep a last-good copy, and serve that copy (marked stale)
when every provider fails.
"""

import logging
import re
from dataclasses import dataclass, field
from datetime import date, datetime, time, timedelta, timezone
from typing import Any, Awaitable, Callable, Dict, List, Optional, Tuple

from ..cache import CacheConfig, CacheManager
from .candles import ICT, MAX_LOOKBACK_DAYS, Timeframe, aggregate_candles, clean_candles
from .chains import Chains, MarketTTL
from .errors import UpstreamError, ValidationError
from .models import CandleSeries, ExchangeRate, SeriesType, TickerType, to_json, utc_iso
from .router import Router
from .store import LastGoodStore
from .symbols import normalise_symbol

logger = logging.getLogger(__name__)

# Rejects rates a feed stopped updating (fmarket's USD/VND history froze in
# 2025) so the chain falls through; covers weekends and holidays.
MAX_EXCHANGE_RATE_AGE = timedelta(days=4)

CACHE_NAMESPACE = "v2"
_CURRENCY = re.compile(r"^[A-Z0-9]{2,10}$")


@dataclass
class Answer:
    data: Any
    provider: Optional[str]
    fallback: List[str] = field(default_factory=list)
    cached: bool = False
    stale: bool = False
    as_of: Optional[str] = None


def _enum(kind, raw: str, label: str):
    value = (raw or "").strip().lower()
    try:
        return kind(value)
    except ValueError:
        raise ValidationError(f"{label} must be one of {', '.join(k.value for k in kind)}") from None


def _symbol(raw: str) -> str:
    symbol = normalise_symbol(raw)
    if not symbol:
        raise ValidationError("symbol is required")
    return symbol


def parse_day(raw: Optional[str], label: str) -> Optional[date]:
    """A calendar day as YYYY-MM-DD (or fmarket's YYYYMMDD)."""
    if raw is None or not raw.strip():
        return None
    for layout in ("%Y-%m-%d", "%Y%m%d"):
        try:
            return datetime.strptime(raw.strip(), layout).date()
        except ValueError:
            continue
    raise ValidationError(f"{label} must be a date as YYYY-MM-DD")


class MarketService:
    def __init__(
        self,
        sources: Dict[str, Any],
        chains: Optional[Chains] = None,
        cache: Optional[CacheManager] = None,
        last_good: Optional[LastGoodStore] = None,
        now: Callable[[], datetime] = lambda: datetime.now(timezone.utc),
    ):
        self.router = Router(sources)
        self.chains = chains or Chains()
        self.cache = cache
        self.last_good = last_good
        self._now = now

    @property
    def sources(self) -> Dict[str, Any]:
        return self.router.sources

    async def aclose(self) -> None:
        for source in self.sources.values():
            await source.aclose()
        if self.last_good is not None:
            await self.last_good.aclose()

    # Core: cache -> route -> last-good

    async def _serve(
        self,
        key: str,
        ttl: Optional[int],
        produce: Callable[[], Awaitable[Tuple[Any, str, List[str]]]],
        skip_cache: bool,
    ) -> Answer:
        """
        Serve `key` from the cache unless skip_cache (which bypasses the read,
        never the write). On a miss, `produce()` routes the request; its
        answer is cached for ttl (None: until 5am ICT) and kept as the
        last-good copy. When every provider fails, the last-good copy is
        served with stale=True, even with skip_cache.
        """

        async def fresh() -> Dict[str, Any]:
            data, provider, fallback = await produce()
            record = {"data": data, "provider": provider, "fallback": fallback, "as_of": utc_iso(self._now())}
            if self.last_good is not None:
                await self.last_good.put(key, record, record["as_of"])
            return record

        try:
            if self.cache is None:
                record, cached = await fresh(), False
            else:
                record, cached = await self.cache.get_or_compute(
                    CACHE_NAMESPACE, {"key": key}, fresh, ttl=CacheConfig.get_ttl(ttl), should_cache=lambda _: True, refresh=skip_cache
                )
        except UpstreamError as cause:
            copy = await self.last_good.get(key) if self.last_good is not None else None
            if copy is None:
                raise
            logger.error("every provider failed for %s, serving the last-good copy as of %s: %s", key, copy.as_of, cause)
            record = copy.record
            return Answer(data=record["data"], provider=record["provider"], fallback=record["fallback"], stale=True, as_of=copy.as_of)
        return Answer(data=record["data"], provider=record["provider"], fallback=record["fallback"], cached=cached, as_of=record["as_of"])

    async def _routed(self, key: str, ttl: Optional[int], operation: str, chain: List[str], method: str, call, skip_cache: bool) -> Answer:
        async def produce():
            routed = await self.router.route(operation, chain, method, call)
            return to_json(routed.value), routed.provider, routed.fallback

        return await self._serve(key, ttl, produce, skip_cache)

    # Capabilities

    async def ticker(self, symbol: str, ticker_type: str, skip_cache: bool = False) -> Answer:
        symbol = _symbol(symbol)
        kind = _enum(TickerType, ticker_type, "type")
        return await self._routed(
            f"ticker:{kind.value}:{symbol}",
            MarketTTL.TICKER,
            f"get ticker {kind.value}/{symbol}",
            self.chains.ticker.get(kind.value, []),
            "get_ticker",
            lambda source: source.get_ticker(symbol, kind),
            skip_cache,
        )

    async def exchange_rate(self, base: str, quote: str, skip_cache: bool = False) -> Answer:
        base, quote = (base or "").strip().upper(), (quote or "").strip().upper()
        if not _CURRENCY.match(base) or not _CURRENCY.match(quote):
            raise ValidationError("from and to must be currency or coin codes such as USD, VND or USDT")
        pair = f"{base}_{quote}"

        async def call(source) -> ExchangeRate:
            rate = await source.get_exchange_rate(base, quote)
            self._check_rate(rate)
            return rate

        return await self._routed(f"fx:{pair}", MarketTTL.FX, f"get exchange rate {pair}", self.chains.fx_chain(pair), "get_exchange_rate", call, skip_cache)

    def _check_rate(self, rate: ExchangeRate) -> None:
        if rate.rate <= 0:
            raise ValueError("returned a non-positive rate")
        try:
            at = datetime.fromisoformat((rate.timestamp or "").replace("Z", "+00:00"))
        except ValueError:
            raise ValueError(f"returned a rate without a readable timestamp {rate.timestamp!r}") from None
        if at.tzinfo is None or self._now() - at > MAX_EXCHANGE_RATE_AGE:
            raise ValueError(f"rate is stale: last updated {rate.timestamp}")

    async def price_series(self, symbol: str, series_type: str, skip_cache: bool = False) -> Answer:
        symbol = _symbol(symbol)
        kind = _enum(SeriesType, series_type, "type")
        return await self._routed(
            f"series:{kind.value}:{symbol}",
            MarketTTL.SERIES,
            f"get price series {kind.value}/{symbol}",
            self.chains.series.get(kind.value, []),
            "get_price_series",
            lambda source: source.get_price_series(symbol, kind),
            skip_cache,
        )

    async def bank_rates(self, skip_cache: bool = False) -> Answer:
        return await self._routed(
            "bank_rates", MarketTTL.BANK_RATES, "get bank interest rates", self.chains.bank_rates, "get_bank_rates", lambda source: source.get_bank_rates(), skip_cache
        )

    async def candles(
        self,
        symbol: str,
        timeframe: str,
        days: Optional[int] = None,
        start: Optional[str] = None,
        end: Optional[str] = None,
        skip_cache: bool = False,
    ) -> Answer:
        """
        About `days` of bars ending now (default per timeframe), or the
        `start`..`end` days in ICT. 4h bars are built from 1h bars.
        """
        symbol = _symbol(symbol)
        tf = Timeframe.parse(timeframe)
        now = self._now()
        start_day, end_day = parse_day(start, "from"), parse_day(end, "to")
        ttl: Optional[int] = MarketTTL.CANDLES
        if start_day or end_day:
            if start_day is None:
                raise ValidationError("from is required with to")
            end_day = end_day or now.astimezone(ICT).date()
            if start_day > end_day:
                raise ValidationError("from must not be after to")
            if (end_day - start_day).days > MAX_LOOKBACK_DAYS:
                raise ValidationError(f"the range must not exceed {MAX_LOOKBACK_DAYS} days")
            window_start = datetime.combine(start_day, time.min, ICT)
            window_end = min(now, datetime.combine(end_day, time.max, ICT))
            key = f"candles:{tf.value}:{symbol}:{start_day}:{end_day}"
            if end_day < now.astimezone(ICT).date():
                ttl = None  # a closed range is final until the next 5am reset
        else:
            if days is not None and days <= 0:
                raise ValidationError("days must be positive")
            days = min(days or tf.analysis_lookback_days, MAX_LOOKBACK_DAYS)
            window_start, window_end = now - timedelta(days=days), now
            key = f"candles:{tf.value}:{symbol}:{days}"

        source_tf = tf.source

        async def call(source) -> CandleSeries:
            series = await source.get_candles(symbol, source_tf, window_start, window_end)
            series.candles = clean_candles(series.candles)
            if not series.candles:
                raise ValueError("returned no candles")
            return series

        async def produce():
            routed = await self.router.route(f"get {tf.value} candles for {symbol}", self.chains.candles, "get_candles", call)
            series: CandleSeries = routed.value
            if tf is not source_tf:
                series.candles = aggregate_candles(series.candles, tf)
            series.symbol, series.timeframe, series.provider = symbol, tf.value, routed.provider
            return to_json(series), routed.provider, routed.fallback

        return await self._serve(key, ttl, produce, skip_cache)

    # fmarket funds and gold. The payload is fmarket's own `data`.

    async def _funds(self, key: str, operation: str, method: str, call, skip_cache: bool) -> Answer:
        return await self._routed(key, MarketTTL.FUNDS, operation, self.chains.funds, method, call, skip_cache)

    async def funds(self, asset_types: List[str], mmf: bool, page: int, page_size: int, skip_cache: bool = False) -> Answer:
        if page < 1 or not 1 <= page_size <= 100:
            raise ValidationError("page must be at least 1 and page_size between 1 and 100")
        asset_types = sorted({a.strip().upper() for a in asset_types if a.strip()})
        key = f"funds:{','.join(asset_types)}:{mmf}:{page}:{page_size}"
        return await self._funds(key, "list funds", "get_funds", lambda s: s.get_funds(asset_types, mmf, page, page_size), skip_cache)

    async def fund(self, code: str, skip_cache: bool = False) -> Answer:
        code = (code or "").strip().upper()
        if not code:
            raise ValidationError("fund code is required")
        return await self._funds(f"fund:{code}", f"get fund {code}", "get_fund", lambda s: s.get_fund(code), skip_cache)

    async def fund_nav(self, fund: str, skip_cache: bool = False) -> Answer:
        """NAV history of a fund given its fmarket product id or its code."""
        fund = (fund or "").strip().upper()
        if not fund:
            raise ValidationError("fund id or code is required")

        async def call(source):
            if fund.isdigit():
                return await source.get_fund_nav(int(fund))
            product = await source.get_fund(fund)
            product_id = product.get("id") if isinstance(product, dict) else None
            if not product_id:
                raise ValueError(f"fmarket has no product id for {fund}")
            return await source.get_fund_nav(int(product_id))

        return await self._funds(f"fund_nav:{fund}", f"get NAV history of {fund}", "get_fund_nav", call, skip_cache)

    async def issuers(self, skip_cache: bool = False) -> Answer:
        return await self._funds("issuers", "list fund issuers", "get_issuers", lambda s: s.get_issuers(), skip_cache)

    async def gold_products(self, skip_cache: bool = False) -> Answer:
        return await self._funds("gold_products", "list gold products", "get_gold_products", lambda s: s.get_gold_products(), skip_cache)

    def _history_window(self, start: Optional[str], end: Optional[str]) -> Tuple[datetime, datetime, str]:
        """Days in ICT; the last 30 days by default."""
        today = self._now().astimezone(ICT).date()
        end_day = parse_day(end, "to") or today
        start_day = parse_day(start, "from") or end_day - timedelta(days=30)
        if start_day > end_day:
            raise ValidationError("from must not be after to")
        return datetime.combine(start_day, time.min, ICT), datetime.combine(end_day, time.min, ICT), f"{start_day}:{end_day}"

    async def gold_history(self, start: Optional[str], end: Optional[str], skip_cache: bool = False) -> Answer:
        window_start, window_end, span = self._history_window(start, end)
        return await self._funds(
            f"gold_history:{span}", "get gold price history", "get_gold_history", lambda s: s.get_gold_history(window_start, window_end), skip_cache
        )

    async def fx_history(self, pair: str, start: Optional[str], end: Optional[str], skip_cache: bool = False) -> Answer:
        normalised = (pair or "USD_VND").strip().upper().replace("/", "_")
        if normalised not in ("USD_VND", "USDVND"):
            raise ValidationError("fx history is available only for USD_VND")
        window_start, window_end, span = self._history_window(start, end)
        return await self._funds(
            f"fx_history:USD_VND:{span}", "get USD/VND rate history", "get_usd_rate_history", lambda s: s.get_usd_rate_history(window_start, window_end), skip_cache
        )
