import asyncio
import threading
import time
from datetime import datetime, timedelta
from unittest.mock import patch

import pandas as pd
from fastapi.testclient import TestClient

from src.cache import CacheConfig, CacheManager, MemoryCache
from src.main import app
from src.modules.shared import deps
from src.modules.company.router import get_company_overview
from src.modules.stocks.router import historical_ttl
from src.ttl_utils import GMT_PLUS_7

client = TestClient(app)


@patch("src.modules.shared.deps.Company")
def test_routes_sharing_params_do_not_share_cache_entries(mock_company):
    company = mock_company.return_value
    company.overview.return_value = pd.DataFrame([{"kind": "overview"}])
    company.events.return_value = pd.DataFrame([{"kind": "events"}])

    overview = client.get("/api/v1/company/overview?symbol=VND").json()
    events = client.get("/api/v1/company/events?symbol=VND").json()

    assert overview["data"][0]["kind"] == "overview"
    assert events["data"][0]["kind"] == "events"
    assert events["cached"] is False


@patch("src.modules.shared.deps.Company")
def test_cache_hits_skip_upstream_and_rate_limit(mock_company):
    overview = mock_company.return_value.overview
    overview.return_value = pd.DataFrame([{"symbol": "VND"}])

    first = client.get("/api/v1/company/overview?symbol=VND").json()
    second = client.get("/api/v1/company/overview?symbol=vnd").json()

    assert first["cached"] is False
    assert second["cached"] is True
    assert overview.call_count == 1
    assert len(deps.get_rate_limiter().requests_minute) == 1


@patch("src.modules.shared.deps.Company")
def test_failed_upstream_calls_are_not_cached(mock_company):
    overview = mock_company.return_value.overview
    overview.side_effect = [RuntimeError("upstream down"), pd.DataFrame([{"symbol": "VND"}])]

    assert client.get("/api/v1/company/overview?symbol=VND").status_code == 500
    response = client.get("/api/v1/company/overview?symbol=VND")
    assert response.status_code == 200
    assert response.json()["cached"] is False


@patch("src.modules.shared.deps.Company")
def test_vnstock_calls_run_off_the_event_loop_thread(mock_company):
    seen = {}

    def overview():
        seen["thread"] = threading.current_thread()
        return pd.DataFrame([{"symbol": "VND"}])

    mock_company.return_value.overview.side_effect = overview

    async def run():
        await get_company_overview(symbol="VND")
        return threading.current_thread()

    loop_thread = asyncio.run(run())
    assert seen["thread"] is not loop_thread


def test_concurrent_misses_share_one_upstream_call():
    manager = CacheManager(redis_url="", redis_token="")
    calls = 0

    async def compute():
        nonlocal calls
        calls += 1
        await asyncio.sleep(0.05)
        return {"success": True, "data": [1]}

    async def run():
        return await asyncio.gather(
            *[
                manager.get_or_compute(
                    "e", {"symbol": "VND"}, compute, ttl=60, should_cache=lambda r: True
                )
                for _ in range(5)
            ]
        )

    results = asyncio.run(run())
    assert calls == 1
    assert all(data == {"success": True, "data": [1]} for data, _ in results)
    # Each caller gets its own copy.
    assert len({id(data) for data, _ in results}) == 5


def test_memory_cache_expires_and_evicts_least_recently_used():
    cache = MemoryCache(max_entries=2)
    cache.set("a", "1", ttl=60)
    cache.set("b", "2", ttl=60)
    cache.get("a")
    cache.set("c", "3", ttl=60)
    assert cache.get("b") is None
    assert cache.get("a") == "1"

    cache.set("d", "4", ttl=60)
    with patch("src.cache.time.monotonic", return_value=time.monotonic() + 61):
        assert cache.get("d") is None


def test_ttls_are_capped_at_the_daily_reset():
    with patch("src.cache.calculate_ttl_until_next_5am", return_value=30):
        assert CacheConfig.get_ttl(CacheConfig.QUOTE_TTL) == 30
        assert CacheConfig.get_ttl(None) == 30
    with patch("src.cache.calculate_ttl_until_next_5am", return_value=5000):
        assert CacheConfig.get_ttl(CacheConfig.QUOTE_TTL) == 60
        assert CacheConfig.get_ttl(CacheConfig.LISTING_TTL) == 3600


def test_historical_ttl_depends_on_whether_the_range_includes_today():
    today = datetime.now(GMT_PLUS_7)
    yesterday = (today - timedelta(days=1)).strftime("%Y-%m-%d")

    assert historical_ttl({"end_date": yesterday}) == CacheConfig.HISTORICAL_CLOSED_TTL
    assert historical_ttl({"end_date": today.strftime("%Y-%m-%d")}) == CacheConfig.HISTORICAL_LIVE_TTL
    assert historical_ttl({}) == CacheConfig.HISTORICAL_LIVE_TTL


def test_sqlite_is_the_l2_when_upstash_is_not_configured(tmp_path):
    manager = CacheManager(redis_url="", redis_token="", sqlite_path=tmp_path / "c.db", enabled=True)
    assert manager.backend.startswith("memory + SQLite")


def test_upstash_wins_over_sqlite_when_configured(tmp_path):
    manager = CacheManager(redis_url="https://x", redis_token="t", sqlite_path=tmp_path / "c.db", enabled=True)
    assert manager.backend == "memory + Upstash Redis"
    assert not (tmp_path / "c.db").exists()


def test_disabled_cache_keeps_memory_only(tmp_path):
    manager = CacheManager(redis_url="", redis_token="", sqlite_path=tmp_path / "c.db", enabled=False)
    assert manager.backend == "memory"
    assert manager.client is None


def test_sqlite_entries_survive_a_restart(tmp_path):
    path = tmp_path / "c.db"

    async def write():
        manager = CacheManager(redis_url="", redis_token="", sqlite_path=path, enabled=True)
        await manager.set("stocks-quote", {"symbol": "FPT"}, {"price": 1}, ttl=60)
        await manager.aclose()

    async def read():
        manager = CacheManager(redis_url="", redis_token="", sqlite_path=path, enabled=True)
        value = await manager.get("stocks-quote", {"symbol": "FPT"})
        # A hit from L2 is promoted into memory with the remaining TTL.
        promoted = len(manager.memory)
        await manager.aclose()
        return value, promoted

    asyncio.run(write())
    assert asyncio.run(read()) == ({"price": 1}, 1)


def test_sqlite_expired_entries_are_misses_and_purged(tmp_path):
    from src.cache import SQLiteCache

    cache = SQLiteCache(tmp_path / "c.db")
    asyncio.run(cache.set("k", "v", ex=60))
    with patch("src.cache.time.time", return_value=time.time() + 61):
        assert asyncio.run(cache.get_with_ttl("k")) == (None, 0)
        cache._purge()
    assert len(cache) == 0


def test_sqlite_flush_and_delete(tmp_path):
    from src.cache import SQLiteCache

    cache = SQLiteCache(tmp_path / "c.db")
    asyncio.run(cache.set("a", "1", ex=60))
    asyncio.run(cache.set("b", "2", ex=60))
    asyncio.run(cache.delete("a"))
    assert asyncio.run(cache.get_with_ttl("a")) == (None, 0)
    assert asyncio.run(cache.get_with_ttl("b"))[0] == "2"
    asyncio.run(cache.flushall())
    assert len(cache) == 0
