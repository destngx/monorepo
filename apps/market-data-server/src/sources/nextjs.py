"""Reading the page props Next.js sites embed in a page as `__NEXT_DATA__` JSON."""

import json
import re
from typing import Any

_NEXT_DATA = re.compile(r'<script id="__NEXT_DATA__"[^>]*>(.*?)</script>', re.S)


def page_props(html: str, source: str) -> dict:
    """`props.pageProps` of the page; raises when the block is missing or unreadable."""
    found = _NEXT_DATA.search(html)
    if not found:
        raise ValueError(f"{source}: the page has no __NEXT_DATA__ block")
    try:
        data: Any = json.loads(found.group(1))
    except ValueError as e:
        raise ValueError(f"{source}: decode __NEXT_DATA__: {e}") from e
    props = ((data.get("props") or {}).get("pageProps")) if isinstance(data, dict) else None
    return props if isinstance(props, dict) else {}
