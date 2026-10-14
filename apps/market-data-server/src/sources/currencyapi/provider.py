"""
fawazahmed0 currency-api: free daily reference rates for fiat and major
coins. The exchange-rate fallback for fmarket, Yahoo and CoinGecko.
"""

import os
from datetime import datetime, timezone
from typing import Optional
from urllib.parse import quote as url_quote

import httpx

from ...market.errors import ValidationError
from ...market.models import ExchangeRate, utc_iso
from ..http import new_client, request_json

# jsDelivr mirror of the latest daily snapshot.
DEFAULT_BASE_URL = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest"


class CurrencyApiSource:
    name = "currencyapi"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        self._client = new_client(base_url or os.getenv("CURRENCY_API_BASE_URL") or DEFAULT_BASE_URL, transport=transport)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await self.get_exchange_rate("USD", "VND")
        return "ok"

    async def get_exchange_rate(self, base: str, quote: str) -> ExchangeRate:
        base, quote = base.strip().lower(), quote.strip().lower()
        if not base or not quote:
            raise ValidationError("currencies are required")
        payload = await request_json(self._client, self.name, "GET", f"/v1/currencies/{url_quote(base, safe='')}.min.json")
        rates = (payload or {}).get(base)
        if not isinstance(rates, dict):
            raise ValueError(f"currencyapi: unreadable {base} rates")
        rate = float(rates.get(quote) or 0)
        if rate <= 0:
            raise ValueError(f"currencyapi has no {base.upper()}/{quote.upper()} rate")
        timestamp = None
        try:
            # The snapshot date, at 00:00Z.
            timestamp = utc_iso(datetime.strptime(str(payload.get("date")), "%Y-%m-%d").replace(tzinfo=timezone.utc))
        except ValueError:
            # Without a date the router rejects the rate as unreadable and moves on.
            pass
        return ExchangeRate(base=base.upper(), quote=quote.upper(), rate=rate, provider=self.name, timestamp=timestamp)
