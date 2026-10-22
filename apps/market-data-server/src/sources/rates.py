"""Reading published savings rates: tables by tenor and product pages."""

from typing import Any, Iterable, Optional, Pattern

from ..market.models import MAX_TENOR_MONTHS


def percent(value: Any) -> Optional[float]:
    """A published rate in percent a year; None when blank, unreadable or not positive."""
    try:
        rate = float(value)
    except (TypeError, ValueError):
        return None
    return rate if rate > 0 else None


def months(value: Any) -> Optional[int]:
    """A tenor in whole months (0 = no-term); None when unreadable or out of range."""
    try:
        tenor = int(str(value).strip())
    except (TypeError, ValueError):
        return None
    return tenor if 0 <= tenor <= MAX_TENOR_MONTHS else None


def stated_rate(texts: Iterable[str], pattern: Pattern[str], source: str) -> float:
    """
    The one rate a product page states in its prose (group 1 of `pattern`,
    a comma decimal accepted). Raises when the page states none, or several
    different ones, so a reworded page fails instead of guessing.
    """
    found = {rate for text in texts for m in pattern.finditer(text) if (rate := percent(m.group(1).replace(",", "."))) is not None}
    if not found:
        raise ValueError(f"{source}: the product page states no rate")
    if len(found) > 1:
        raise ValueError(f"{source}: the product page states several rates: {', '.join(f'{r:g}%' for r in sorted(found))}")
    return found.pop()
