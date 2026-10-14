"""CoinGecko: crypto prices in USD, and coins quoted in USD or VND (USDT_VND)."""

import asyncio
import os
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Dict, Optional

import httpx

from ...market.errors import Unsupported, ValidationError
from ...market.models import ExchangeRate, RateBasis, Ticker, TickerType, utc_iso
from ..http import new_client, request_json

DEFAULT_BASE_URL = "https://api.coingecko.com/api/v3"

# Common symbols pinned to CoinGecko ids so they never depend on search
# ranking, where symbols collide (many tokens are called "SOL").
KNOWN_COINS = {
    "BTC": "bitcoin", "ETH": "ethereum", "USDT": "tether", "USDC": "usd-coin", "FDUSD": "first-digital-usd",
    "DAI": "dai", "BNB": "binancecoin", "SOL": "solana", "XRP": "ripple", "ADA": "cardano", "DOGE": "dogecoin",
    "TON": "the-open-network", "TRX": "tron", "DOT": "polkadot", "POL": "polygon-ecosystem-token",
    "LINK": "chainlink", "AVAX": "avalanche-2", "LTC": "litecoin", "SUI": "sui", "ARB": "arbitrum", "OP": "optimism",
}  # fmt: skip


@dataclass
class _Quote:
    value: float
    updated_at: Optional[str]


class CoinGeckoSource:
    name = "coingecko"

    def __init__(self, base_url: Optional[str] = None, api_key: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        key = (api_key if api_key is not None else os.getenv("COINGECKO_API_KEY") or "").strip()
        # The optional demo key raises the rate limit; it is never logged.
        headers = {"x-cg-demo-api-key": key} if key else {}
        self._client = new_client(base_url or os.getenv("COINGECKO_BASE_URL") or DEFAULT_BASE_URL, headers, transport)
        self._resolved: Dict[str, str] = {}
        self._lock = asyncio.Lock()

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await request_json(self._client, self.name, "GET", "/ping")
        return "ok"

    async def get_ticker(self, symbol: str, ticker_type: TickerType) -> Ticker:
        if ticker_type is not TickerType.CRYPTO:
            raise Unsupported(f"coingecko serves only {TickerType.CRYPTO.value} tickers")
        quote = await self._price(symbol, "usd")
        return Ticker(symbol=symbol.upper(), type=ticker_type, provider=self.name, currency="USD", price=quote.value, timestamp=quote.updated_at)

    async def get_exchange_rate(self, base: str, quote: str) -> ExchangeRate:
        """A coin (base) quoted in USD or VND, e.g. USDT_VND."""
        currency = quote.lower()
        if currency not in ("usd", "vnd"):
            raise Unsupported("coingecko quotes coins only in USD or VND")
        price = await self._price(base, currency)
        return ExchangeRate(base=base.upper(), quote=currency.upper(), rate=price.value, provider=self.name, timestamp=price.updated_at, basis=RateBasis.CRYPTO_MARKET)

    async def _price(self, symbol: str, currency: str) -> _Quote:
        coin = await self._coin_id(symbol)
        payload = await request_json(
            self._client, self.name, "GET", "/simple/price", params={"ids": coin, "vs_currencies": currency, "include_last_updated_at": "true"}
        )
        entry = (payload or {}).get(coin) or {}
        value = float(entry.get(currency) or 0)
        if value <= 0:
            raise ValueError(f"coingecko has no {currency.upper()} price for {coin}")
        updated = entry.get("last_updated_at") or 0
        return _Quote(value=value, updated_at=utc_iso(datetime.fromtimestamp(updated, tz=timezone.utc)) if updated > 0 else None)

    async def _coin_id(self, symbol: str) -> str:
        """Pinned id first, then the exact-symbol search hit with the best market-cap rank (memoised)."""
        symbol = symbol.strip().upper()
        if not symbol:
            raise ValidationError("coin symbol is required")
        if symbol in KNOWN_COINS:
            return KNOWN_COINS[symbol]
        async with self._lock:
            if symbol in self._resolved:
                return self._resolved[symbol]
            payload = await request_json(self._client, self.name, "GET", "/search", params={"query": symbol})
            ranked = [
                coin
                for coin in (payload or {}).get("coins") or []
                if str(coin.get("symbol", "")).upper() == symbol and (coin.get("market_cap_rank") or 0) > 0
            ]
            if not ranked:
                raise ValueError(f"coingecko has no ranked coin with symbol {symbol}")
            coin = min(ranked, key=lambda c: c["market_cap_rank"])["id"]
            self._resolved[symbol] = coin
            return coin
