"""
fmarket (api.fmarket.vn): open-ended fund NAVs, SJC and world gold, the
USD/VND reference rate and the savings rate table.
"""

import os
from datetime import datetime, timedelta, timezone
from typing import Any, Callable, Dict, List, Optional
from urllib.parse import quote as url_quote

import httpx

from ...market.candles import ICT, Timeframe, aggregate_candles, clean_candles
from ...market.errors import NotFound, Unsupported, ValidationError
from ...market.models import (
    BankRate,
    Candle,
    CandleSeries,
    ExchangeRate,
    PricePoint,
    PriceSeries,
    SeriesType,
    Ticker,
    TickerType,
    utc_iso,
)
from ...market.symbols import SJC, XAUUSD
from ..http import StatusError, new_client, request_json

DEFAULT_BASE_URL = "https://api.fmarket.vn"

# fmarket rejects requests that do not look like its own web app.
HEADERS = {
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.7; rv:123.0) Gecko/20100101 Firefox/123.0",
    "Accept": "application/json, text/plain, */*",
    "Origin": "https://fmarket.vn",
    "Referer": "https://fmarket.vn/",
    "F-Language": "vi",
}


def fmarket_date(at: datetime) -> str:
    """
    A day the way fmarket's history endpoints read it. Any other layout
    (2026-01-02 included) is ignored and fmarket answers with a stale default
    window instead of an error.
    """
    return at.astimezone(ICT).strftime("%Y%m%d")


def _from_ms(ms: int) -> datetime:
    return datetime.fromtimestamp(ms / 1000, tz=timezone.utc)


def _oldest_first(rows: List[dict]) -> List[dict]:
    # Rate and gold histories arrive newest first.
    return sorted(rows, key=lambda row: row.get("reportDate") or 0)


def _number(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


# fmarket's error code for an unknown product code (HTTP 400).
INVALID_PRODUCT_CODE = 40065


class FmarketSource:
    name = "fmarket"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None, now: Callable[[], datetime] = lambda: datetime.now(timezone.utc)):
        self._client = new_client(base_url or os.getenv("FMARKET_BASE_URL") or DEFAULT_BASE_URL, HEADERS, transport)
        self._now = now

    async def aclose(self) -> None:
        await self._client.aclose()

    async def _payload(self, method: str, path: str, body: Any = None) -> Dict[str, Any]:
        payload = await request_json(self._client, self.name, method, path, json=body)
        if not isinstance(payload, dict) or "data" not in payload:
            raise ValueError(f"fmarket answered {path} without a data field")
        return payload

    async def _data(self, method: str, path: str, body: Any = None) -> Any:
        return (await self._payload(method, path, body))["data"]

    def _last_days(self, days: int) -> Dict[str, Any]:
        now = self._now()
        return {"fromDate": fmarket_date(now - timedelta(days=days)), "toDate": fmarket_date(now), "isAllData": False}

    async def health(self) -> str:
        payload = await request_json(self._client, self.name, "GET", "/res/bank-interest-rate")
        status = payload.get("status") if isinstance(payload, dict) else None
        try:
            return "degraded" if status is not None and int(status) >= 400 else "ok"
        except (TypeError, ValueError):
            return "ok"

    # Capabilities

    async def get_ticker(self, symbol: str, ticker_type: TickerType) -> Ticker:
        if ticker_type is TickerType.IFC:
            product = await self.get_fund(symbol)
            nav = _number(product.get("nav")) if isinstance(product, dict) else 0.0
            if nav <= 0:
                raise ValueError(f"fmarket has no NAV for {symbol}")
            update_at = ((product.get("productNavChange") or {}).get("updateAt")) or 0
            return Ticker(
                symbol=product.get("code") or symbol,
                type=ticker_type,
                provider=self.name,
                currency="VND",
                price=nav,
                timestamp=utc_iso(_from_ms(update_at)) if update_at > 0 else None,
            )
        if ticker_type is TickerType.GOLD:
            gold = (await self._gold_series()).data
            if symbol.upper() == "XAU":
                return Ticker(symbol="XAU", type=ticker_type, provider=self.name, currency="USD", price=gold["price"], timestamp=gold["timestamp"])
            return Ticker(symbol=symbol, type=ticker_type, provider=self.name, currency="VND", bid=gold["bidSjc"], ask=gold["askSjc"], timestamp=gold["timestamp"])
        raise Unsupported(f"fmarket does not serve {ticker_type.value} tickers")

    async def get_price_series(self, symbol: str, series_type: SeriesType) -> PriceSeries:
        if series_type is SeriesType.GOLD_USD:
            return await self._gold_series()
        if series_type is SeriesType.FOREX:
            return await self._forex_series(symbol)
        raise Unsupported(f"fmarket serves only {SeriesType.GOLD_USD.value} and {SeriesType.FOREX.value} price series")

    async def _forex_series(self, symbol: str) -> PriceSeries:
        """The USD/VND sell rate over the last 30 days."""
        pair = symbol.strip().upper().replace("/", "_")
        if pair not in ("", "USD_VND", "USDVND"):
            raise Unsupported("fmarket has a forex series only for USD_VND")
        rows = _oldest_first(await self._data("POST", "/res/get-usd-rate-history", self._last_days(30)))
        points = [PricePoint(price=row["rateSellUsd"], timestamp=utc_iso(_from_ms(row["reportDate"]))) for row in rows if _number(row.get("rateSellUsd")) > 0]
        if not points:
            raise ValueError("fmarket returned an empty USD/VND series")
        return PriceSeries(provider=self.name, series_type=SeriesType.FOREX, currency="VND", data={"pair": "USD_VND", "points": points})

    async def _gold_series(self) -> PriceSeries:
        rows = _oldest_first(await self._data("POST", "/res/get-price-gold-history", self._last_days(30)))
        if not rows:
            raise ValueError("fmarket returned an empty gold price series")
        points = [PricePoint(price=_number(row.get("price")), timestamp=utc_iso(_from_ms(row["reportDate"]))) for row in rows]
        last = rows[-1]
        return PriceSeries(
            provider=self.name,
            series_type=SeriesType.GOLD_USD,
            currency="USD",
            data={
                "price": _number(last.get("price")),
                "points": points,
                "bidSjc": _number(last.get("bidSjc")),
                "askSjc": _number(last.get("askSjc")),
                "timestamp": utc_iso(_from_ms(last["reportDate"])),
            },
        )

    async def get_candles(self, symbol: str, timeframe: Timeframe, start: datetime, end: datetime) -> CandleSeries:
        """
        Daily (and weekly, aggregated) gold history: SJC bars use the SJC sell
        price in VND, XAUUSD the world price in USD per ounce. fmarket
        publishes one quote a day, so each bar is a single close.
        """
        if symbol not in (SJC, XAUUSD):
            raise Unsupported(f"fmarket has no {symbol} history")
        if timeframe not in (Timeframe.D1, Timeframe.W1):
            raise Unsupported("fmarket gold history is daily")
        rows = await self._data("POST", "/res/get-price-gold-history", {"fromDate": fmarket_date(start), "toDate": fmarket_date(end), "isAllData": False})
        field = "askSjc" if symbol == SJC else "price"
        candles = clean_candles([Candle(time=_from_ms(row["reportDate"]), close=_number(row.get(field))) for row in rows if row.get("reportDate")])
        if timeframe is Timeframe.W1:
            candles = aggregate_candles(candles, Timeframe.W1)
        return CandleSeries(symbol=symbol, timeframe=timeframe.value, provider=self.name, currency="VND" if symbol == SJC else "USD", candles=candles)

    async def get_exchange_rate(self, base: str, quote: str) -> ExchangeRate:
        if base.upper() != "USD" or quote.upper() != "VND":
            raise Unsupported("fmarket quotes only USD_VND")
        rows = [row for row in await self._data("POST", "/res/get-usd-rate-history", self._last_days(7)) if _number(row.get("rateSellUsd")) > 0]
        if not rows:
            raise ValueError("fmarket returned no exchange-rate data")
        last = _oldest_first(rows)[-1]
        return ExchangeRate(base="USD", quote="VND", rate=_number(last["rateSellUsd"]), provider=self.name, timestamp=utc_iso(_from_ms(last["reportDate"])))

    async def get_bank_rates(self) -> List[BankRate]:
        """
        fmarket publishes one savings rate per bank without a tenor, so tenor
        stays empty rather than guessed; updated_at is the table's date.
        """
        data = await self._data("GET", "/res/bank-interest-rate")
        updated_at = None
        try:
            updated_at = datetime.strptime(str(data.get("updateAt") or ""), "%d/%m/%Y").strftime("%Y-%m-%d")
        except ValueError:
            pass
        rates = []
        for item in data.get("bankList") or []:
            rate = _number(str(item.get("value") or "").strip())
            if rate > 0:
                rates.append(BankRate(bank=item.get("name") or item.get("key") or "", rate=rate, currency="VND", updated_at=updated_at))
        return rates

    # Funds and gold products. These return fmarket's `data` payload as is;
    # wealth-management's UI parses it today.

    async def get_funds(self, asset_types: List[str], mmf: bool, page: int, page_size: int) -> Any:
        return await self._data(
            "POST",
            "/res/products/filter",
            {
                "types": ["NEW_FUND", "TRADING_FUND"],
                "sortOrder": "DESC",
                "sortField": "navTo12Months",
                "isIpo": False,
                "isMMFFund": mmf,
                "fundAssetTypes": [] if mmf else (asset_types or ["STOCK"]),
                "page": page,
                "pageSize": page_size,
            },
        )

    async def get_fund(self, code: str) -> Any:
        code = code.strip()
        if not code:
            raise ValidationError("fund code is required")
        try:
            return await self._data("GET", "/home/product/" + url_quote(code, safe=""))
        except StatusError as e:
            if e.status == 400 and e.code == INVALID_PRODUCT_CODE:
                raise NotFound(f"fmarket has no fund {code}") from e
            raise

    async def get_fund_nav(self, product_id: int) -> Any:
        # isAllData returns the full history, but fmarket still rejects the
        # request ("invalid date format") unless a navPeriod is present.
        body = {"isAllData": 1, "productId": product_id, "navPeriod": "navToBeginning"}
        return await self._data("POST", "/res/product/get-nav-history", body)

    async def get_issuers(self) -> Any:
        return await self._data("POST", "/res/issuers", {})

    async def get_gold_products(self) -> Any:
        return await self._data(
            "POST",
            "/res/products/filter",
            {"types": ["GOLD"], "issuerIds": [], "page": 1, "pageSize": 100, "fundAssetTypes": [], "bondRemainPeriods": [], "searchField": ""},
        )

    async def get_gold_history(self, start: datetime, end: datetime) -> Dict[str, Any]:
        """
        `rows` are the daily prices, newest first. `summary` is fmarket's
        `extra` block: today's ring gold bid/ask, USD rate, world price in VND
        and day changes (fmarket names ring prices from the buyer's side).
        """
        body = {"fromDate": fmarket_date(start), "toDate": fmarket_date(end), "isAllData": False}
        payload = await self._payload("POST", "/res/get-price-gold-history", body)
        return {"rows": payload["data"], "summary": payload.get("extra") or {}}

    async def get_usd_rate_history(self, start: datetime, end: datetime) -> Any:
        return await self._data("POST", "/res/get-usd-rate-history", {"fromDate": fmarket_date(start), "toDate": fmarket_date(end), "isAllData": False})
