"""
MoMo Túi Thần Tài (momo.vn/tui-than-tai): money in the MoMo app that earns
a daily compounding return and can be spent or withdrawn at any time. It is
not a deposit: the user signs a business cooperation contract with Finsight,
whose money is invested by Thiên Việt fund management (TVAM) and held in
custody at Vietcombank, and the page calls it a product with financial
risk. Withdrawals usually reach the MoMo wallet in 5 minutes, at most 16
working hours. Money moved in Monday to Thursday starts earning the next
working day, Friday to Sunday the next Monday ("Nguyên tắc tính lợi
nhuận").

MoMo has no rate API. The page embeds its CMS blocks as Next.js page props
(`__NEXT_DATA__`) and states the rate in prose, several times ("đến 4%/năm",
"tỷ suất sinh lời cố định 4%/năm"); every statement must agree. The page has
no update date.
"""

import html
import os
import re
import unicodedata
from typing import Any, Iterator, List, Optional

import httpx

from ...market.models import EarningStart, FlexibleSaving, SavingsBacking
from ..http import new_client, request_text
from ..nextjs import page_props
from ..rates import stated_rate

DEFAULT_BASE_URL = "https://www.momo.vn"
PAGE_PATH = "/tui-than-tai"
PRODUCT_ID = "momo:tui-than-tai"
PRODUCT = "Túi Thần Tài"
WITHDRAWAL_MAX_WORKING_HOURS = 16

_TAG = re.compile(r"<[^>]+>")
_SPACE = re.compile(r"\s+")
_RATE = re.compile(r"(?:tỷ suất sinh lời(?: cố định| lên đến)?|tiền lời nhận về mỗi ngày, đến)\s+(\d+(?:[.,]\d+)?)\s*%\s*/\s*năm", re.I)


def _plain(text: str) -> str:
    """CMS HTML as plain text. Some blocks are entity-escaped twice, and some
    letters are written with combining accents ("ô" + U+0301), so the text is
    unescaped twice and composed (NFC)."""
    text = html.unescape(html.unescape(text))
    return unicodedata.normalize("NFC", _SPACE.sub(" ", html.unescape(_TAG.sub(" ", text))))


def _strings(value: Any) -> Iterator[str]:
    if isinstance(value, str):
        yield _plain(value)
    elif isinstance(value, dict):
        for item in value.values():
            yield from _strings(item)
    elif isinstance(value, list):
        for item in value:
            yield from _strings(item)


class MomoSource:
    name = "momo"

    def __init__(self, base_url: Optional[str] = None, transport: Optional[httpx.AsyncBaseTransport] = None):
        self._base_url = (base_url or os.getenv("MOMO_BASE_URL") or DEFAULT_BASE_URL).strip().rstrip("/")
        self._client = new_client(self._base_url, {"Accept": "text/html"}, transport)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> str:
        await self.get_flexible_savings()
        return "ok"

    async def get_flexible_savings(self) -> List[FlexibleSaving]:
        blocks = page_props(await request_text(self._client, self.name, PAGE_PATH), self.name).get("dataServiceListBlock")
        if not blocks:
            raise ValueError("momo: the product page has no content blocks")
        return [
            FlexibleSaving(
                id=PRODUCT_ID,
                provider="MoMo",
                product=PRODUCT,
                rate=stated_rate(_strings(blocks), _RATE, self.name),
                backing=SavingsBacking.BUSINESS_COOPERATION,
                earning_starts=EarningStart.NEXT_WORKING_DAY,
                url=self._base_url + PAGE_PATH,
                withdrawal_max_working_hours=WITHDRAWAL_MAX_WORKING_HOURS,
            )
        ]
