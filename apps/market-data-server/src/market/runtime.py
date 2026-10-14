"""Builds the market service and its sources once per process."""

import logging
from typing import Optional

from ..modules.shared.deps import get_cache_manager
from ..sources.algolab import AlgolabSource
from ..sources.coingecko import CoinGeckoSource
from ..sources.currencyapi import CurrencyApiSource
from ..sources.fmarket import FmarketSource
from ..sources.vnstock import VnstockSource
from ..sources.yahoo import YahooSource
from .chains import load_chains
from .service import MarketService
from .store import LastGoodStore

logger = logging.getLogger(__name__)

_SERVICE: Optional[MarketService] = None


def build_market_service() -> MarketService:
    sources = [VnstockSource(), FmarketSource(), YahooSource(), CoinGeckoSource(), CurrencyApiSource(), AlgolabSource()]
    store = LastGoodStore()
    logger.info("✓ Market sources: %s; last-good store at %s", ", ".join(s.name for s in sources), store.path)
    return MarketService({s.name: s for s in sources}, chains=load_chains(), cache=get_cache_manager(), last_good=store)


def get_market_service() -> MarketService:
    """FastAPI dependency; tests override it with a service over fake sources."""
    global _SERVICE
    if _SERVICE is None:
        _SERVICE = build_market_service()
    return _SERVICE


async def close_market_service() -> None:
    global _SERVICE
    if _SERVICE is not None:
        await _SERVICE.aclose()
        _SERVICE = None
