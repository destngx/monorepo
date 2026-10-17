"""
Simplize (simplize.vn/lai-suat-ngan-hang): savings rates by tenor with a
no-term ("demand") rate and a daily date per bank. The page has no public
JSON API; the table is the Next.js page props embedded in the page as
`__NEXT_DATA__` JSON, which is read instead of the rendered HTML. The
source does not say whether a rate is online or at the counter.
"""

import json
import os
import re
from datetime import datetime, timezone
from typing import Any, List, Optional

import httpx

from ...market.candles import ICT
from ...market.models import NO_TERM, BankRate
from ..http import new_client, request_text
from ..rates import months, percent

DEFAULT_BASE_URL = "https://simplize.vn"
PAGE_PATH = "/lai-suat-ngan-hang"

_NEXT_DATA = re.compile(r'<script id="__NEXT_DATA__"[^>]*>(.*?)</script>', re.S)
# maturity1m, maturity12m, ...
_MATURITY = re.compile(r"^maturity(\d+)m$")
SAVING_RATE = "saving_rate"


def _day(ms: Any) -> Optional[str]:
    """A table date (epoch milliseconds) as its day in Vietnam."""
    try:
        return datetime.fromtimestamp(int(ms) / 1000, tz=timezone.utc).astimezone(ICT).strftime("%Y-%m-%d")
    except (TypeError, ValueError, OverflowError, OSError):
        return None


class SimplizeSource:
    name = "simplize"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        self._client = new_client(base_url or os.getenv("SIMPLIZE_BASE_URL") or DEFAULT_BASE_URL, {"Accept": "text/html"}, transport)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await self.get_bank_rates()
        return "ok"

    async def _page_props(self) -> Any:
        html = await request_text(self._client, self.name, PAGE_PATH)
        found = _NEXT_DATA.search(html)
        if not found:
            raise ValueError("simplize: the rate page has no __NEXT_DATA__ block")
        try:
            data = json.loads(found.group(1))
        except ValueError as e:
            raise ValueError(f"simplize: decode __NEXT_DATA__: {e}") from e
        return ((data.get("props") or {}).get("pageProps") or {}) if isinstance(data, dict) else {}

    async def get_bank_rates(self) -> List[BankRate]:
        rows = (await self._page_props()).get("interestRate")
        rates = []
        for row in rows if isinstance(rows, list) else []:
            if not isinstance(row, dict) or row.get("instrumentType", SAVING_RATE) != SAVING_RATE:
                continue
            name = str(row.get("name") or "").strip()
            if not name:
                continue
            updated_at = _day(row.get("date"))
            tenors = [(NO_TERM, row.get("demand"))]
            for key, value in row.items():
                matched = _MATURITY.match(key)
                if matched:
                    tenors.append((months(matched.group(1)), value))
            for tenor, value in sorted(((t, v) for t, v in tenors if t is not None), key=lambda pair: pair[0]):
                rate = percent(value)
                if rate is not None:
                    rates.append(BankRate.by_tenor(name, tenor, rate, updated_at=updated_at))
        if not rates:
            raise ValueError("simplize published no savings rates")
        return rates
