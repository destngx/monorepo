import os

# Set before `src` is imported: tests never touch a shared L2 cache from .env.local.
os.environ["CACHE_ENABLED"] = "false"
os.environ["MARKET_DB_PATH"] = ":memory:"

import pytest
import src.main  # noqa: F401  (initialises the shared services)
from src.modules.shared import deps


@pytest.fixture(autouse=True)
def isolate_shared_state():
    """Each test starts with an empty cache and a fresh rate-limit window."""
    deps.get_cache_manager().memory.clear()
    limiter = deps.get_rate_limiter()
    limiter.requests_minute.clear()
    limiter.requests_day.clear()
    limiter.backoff_until = None
    yield
