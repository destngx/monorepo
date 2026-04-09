"""
Shared dependencies for modules.
"""

from typing import Optional
from src.adapters.cache import MockRedisAdapter


class Services:
    def __init__(self):
        self.cache = MockRedisAdapter()


_services: Optional[Services] = None


def init_services() -> None:
    global _services
    _services = Services()


def get_services() -> Services:
    if _services is None:
        init_services()
    return _services


def get_cache() -> MockRedisAdapter:
    return get_services().cache
