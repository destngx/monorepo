"""Market symbols: canonical form and the instruments with special handling."""

VNINDEX = "VNINDEX"
VN30 = "VN30"
USDVND = "USDVND"
SJC = "SJC"
XAUUSD = "XAUUSD"
VIX = "VIX"

# Instruments that are not HOSE/HNX/UPCoM listings.
NON_VN_LISTINGS = frozenset({USDVND, SJC, XAUUSD, VIX})


def normalise_symbol(symbol: str) -> str:
    """Canonical form used in cache keys and provider calls: trimmed, upper case, no "^" prefix."""
    return (symbol or "").strip().upper().removeprefix("^")


def is_vn_listing(symbol: str) -> bool:
    """Whether symbol is a VN ticker or index, as opposed to FX, gold or a global index."""
    return normalise_symbol(symbol) not in NON_VN_LISTINGS
