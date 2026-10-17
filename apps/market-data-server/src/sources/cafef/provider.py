"""
CafeF (cafef.vn/du-lieu/lai-suat-ngan-hang.chn): the savings rate table by
tenor, served as a static JSON file on CafeF's CDN. The page states the
rates are counter rates paid at maturity, without online or promotional
offers. It publishes no-term (0T) and 1 to 24 months, and no update date.
"""

import os
from typing import List, Optional

import httpx

from ...market.models import BankRate, RateChannel
from ..http import new_client, request_json
from ..rates import months, percent

DEFAULT_BASE_URL = "https://cafefnew.mediacdn.vn"
TABLE_PATH = "/Images/Uploaded/DuLieuDownload/Liveboard/all_banks_interest_rates.json"


class CafefSource:
    name = "cafef"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        self._client = new_client(base_url or os.getenv("CAFEF_BASE_URL") or DEFAULT_BASE_URL, transport=transport)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await self.get_bank_rates()
        return "ok"

    async def get_bank_rates(self) -> List[BankRate]:
        payload = await request_json(self._client, self.name, "GET", TABLE_PATH)
        banks = payload.get("Data") if isinstance(payload, dict) else None
        if not isinstance(banks, list):
            raise ValueError("cafef answered the rate table without a Data list")
        rates = []
        for bank in banks:
            name = str((bank or {}).get("name") or "").strip()
            if not name:
                continue
            for row in bank.get("interestRates") or []:
                tenor, rate = months(row.get("deposit")), percent(row.get("value"))
                if tenor is not None and rate is not None:
                    rates.append(BankRate.by_tenor(name, tenor, rate, channel=RateChannel.COUNTER))
        if not rates:
            raise ValueError("cafef published no savings rates")
        return rates
