"""Market symbols: canonical form and the instruments with special handling."""

import re

VNINDEX = "VNINDEX"
VN30 = "VN30"
USDVND = "USDVND"
SJC = "SJC"
XAUUSD = "XAUUSD"
VIX = "VIX"

# Instruments that are not HOSE/HNX/UPCoM listings.
NON_VN_LISTINGS = frozenset({USDVND, SJC, XAUUSD, VIX})

# HOSE/HNX/UPCoM stocks (FPT), ETFs (E1VFVN30), warrants (CFPT2401),
# futures (VN30F1M) and indices (UPCOMINDEX): 3-10 letters and digits.
_VN_LISTING = re.compile(r"^[A-Z0-9]{3,10}$")


def normalise_symbol(symbol: str) -> str:
    """Canonical form used in cache keys and provider calls: trimmed, upper case, no "^" prefix."""
    return (symbol or "").strip().upper().removeprefix("^")


def is_vn_listing(symbol: str) -> bool:
    """
    Whether symbol can be a VN ticker or index, as opposed to FX, gold or a
    Yahoo-style global symbol (BTC-USD, EURUSD=X), which VN sources skip
    instead of spending quota on.
    """
    symbol = normalise_symbol(symbol)
    return bool(_VN_LISTING.match(symbol)) and symbol not in NON_VN_LISTINGS
