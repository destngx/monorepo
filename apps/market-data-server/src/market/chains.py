"""
Provider chains and cache lifetimes per capability (MARKET_PLATFORM.md
section 3.3). Chains are configuration: `MARKET_CHAINS` (JSON) replaces
any top-level entry, e.g. {"candles": ["yahoo", "vnstock"]}.
"""

import copy
import json
import os
from dataclasses import dataclass, field, fields
from typing import Dict, List, Optional

# The fallback chain for exchange-rate pairs without their own entry.
ANY_PAIR = "*"


@dataclass
class Chains:
    ticker: Dict[str, List[str]] = field(
        default_factory=lambda: {
            "equity": ["vnstock", "fmarket", "algolab"],
            "ifc": ["fmarket", "vnstock"],
            "gold": ["fmarket"],
            "crypto": ["coingecko", "yahoo"],
        }
    )
    fx: Dict[str, List[str]] = field(
        default_factory=lambda: {
            "USD_VND": ["fmarket", "yahoo", "currencyapi"],
            "USDT_VND": ["coingecko", "currencyapi"],
            ANY_PAIR: ["yahoo", "currencyapi"],
        }
    )
    series: Dict[str, List[str]] = field(
        default_factory=lambda: {
            "equity": ["vnstock", "fmarket", "algolab"],
            "gold_usd": ["fmarket"],
            "forex": ["yahoo", "fmarket"],
        }
    )
    candles: List[str] = field(default_factory=lambda: ["vnstock", "yahoo", "fmarket", "algolab"])
    # Rates by tenor first; fmarket's one headline rate per bank (no tenor)
    # is the last fallback and is skipped by tenor filters.
    bank_rates: List[str] = field(default_factory=lambda: ["cafef", "simplize", "fmarket"])
    # Not a chain: every listed source answers for its own products
    # (Zalopay's earning balance is not a fallback for another wallet's).
    flexible_savings: List[str] = field(default_factory=lambda: ["zalopay", "momo"])
    # Fund NAVs, gold products and fmarket's rate histories.
    funds: List[str] = field(default_factory=lambda: ["fmarket"])

    def fx_chain(self, pair: str) -> List[str]:
        return self.fx.get(pair) or self.fx.get(ANY_PAIR, [])


def load_chains(raw: Optional[str] = None) -> Chains:
    chains = Chains()
    raw = raw if raw is not None else os.getenv("MARKET_CHAINS")
    if not raw:
        return chains
    overrides = json.loads(raw)
    known = {f.name for f in fields(Chains)}
    unknown = set(overrides) - known
    if unknown:
        raise ValueError(f"MARKET_CHAINS has unknown capabilities: {', '.join(sorted(unknown))}")
    for name, value in overrides.items():
        if type(value) is not type(getattr(chains, name)):
            raise ValueError(f"MARKET_CHAINS.{name} must be a {type(getattr(chains, name)).__name__}")
        setattr(chains, name, copy.deepcopy(value))
    return chains


class MarketTTL:
    """Seconds an answer stays cached; each is also capped at the next 5am ICT reset."""

    TICKER = 60
    FX = 600
    SERIES = 300
    CANDLES = 300
    # Banks change rates at most daily, at any time of day.
    BANK_RATES = 6 * 3600
    # Providers edit a published product rate by hand, a few times a year.
    FLEXIBLE_SAVINGS = 6 * 3600
    FUNDS = 60
