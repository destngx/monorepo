"""
Zalopay Số dư sinh lời (zalopay.vn/dich-vu/so-du-sinh-loi): an e-wallet
balance that earns daily compounding interest and can be spent or
withdrawn at any time; the page says the money is held as a bank deposit.
Money moved in Monday to Thursday starts earning the next working day,
Friday to Sunday the next Monday, before a public holiday the first working
day after it (the page's "Nguyên tắc tính lời"). Zalopay has no
rate API: its editors set the rate in the CMS, and the page embeds that
content as Next.js page props (`__NEXT_DATA__`). The rate is the feature
line "Sinh lời 4.5% từ khoản chi tiêu hằng tháng". The page's SEO title
carries its own copy of the rate that editors do not keep in step (it said
4.7% while the feature line said 4.5%), so the title is never read.
"""

import os
import re
from datetime import datetime, timezone
from typing import Any, Iterator, List, Optional

import httpx

from ...market.candles import ICT
from ...market.models import EarningStart, FlexibleSaving, SavingsBacking
from ..http import new_client, request_text
from ..nextjs import page_props
from ..rates import stated_rate

DEFAULT_BASE_URL = "https://zalopay.vn"
PAGE_PATH = "/dich-vu/so-du-sinh-loi"
PRODUCT_ID = "zalopay:so-du-sinh-loi"
PRODUCT = "Số dư sinh lời"

# "Sinh lời 4.5% từ khoản chi tiêu hằng tháng"; a comma decimal is accepted.
_RATE = re.compile(r"\bsinh lời\s+(\d+(?:[.,]\d+)?)\s*%", re.I)


def _day(raw: Any) -> Optional[str]:
    """A CMS timestamp (ISO 8601, UTC; Strapi writes a Z, one without an offset is read as UTC too) as its day in Vietnam."""
    try:
        at = datetime.fromisoformat(str(raw).replace("Z", "+00:00"))
    except (TypeError, ValueError):
        return None
    return (at if at.tzinfo else at.replace(tzinfo=timezone.utc)).astimezone(ICT).strftime("%Y-%m-%d")


def _titles(items: Any) -> Iterator[str]:
    """The titles of the page's content blocks (`items[].content[]`)."""
    for item in items if isinstance(items, list) else []:
        for block in (item or {}).get("content") or [] if isinstance(item, dict) else []:
            if isinstance(block, dict) and isinstance(block.get("title"), str):
                yield block["title"]


class ZalopaySource:
    name = "zalopay"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        self._base_url = (base_url or os.getenv("ZALOPAY_BASE_URL") or DEFAULT_BASE_URL).strip().rstrip("/")
        self._client = new_client(self._base_url, {"Accept": "text/html"}, transport)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await self.get_flexible_savings()
        return "ok"

    async def _service_detail(self) -> dict:
        detail = page_props(await request_text(self._client, self.name, PAGE_PATH), self.name).get("service_detail")
        if not isinstance(detail, dict):
            raise ValueError("zalopay: the product page has no service_detail")
        return detail

    async def get_flexible_savings(self) -> List[FlexibleSaving]:
        detail = await self._service_detail()
        rate = stated_rate(_titles(detail.get("items")), _RATE, self.name)
        return [
            FlexibleSaving(
                id=PRODUCT_ID,
                provider="Zalopay",
                product=PRODUCT,
                rate=rate,
                backing=SavingsBacking.DEPOSIT,
                earning_starts=EarningStart.NEXT_WORKING_DAY,
                url=self._base_url + PAGE_PATH,
                updated_at=_day(detail.get("updated_at")),
            )
        ]
