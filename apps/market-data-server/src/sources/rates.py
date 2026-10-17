"""Reading savings-rate tables published by tenor."""

from typing import Any, Optional

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
