import os

import pytest

from src.adapters.cache import RedisAdapter


def test_live_redis_roundtrip():
    if not os.getenv("UPSTASH_REDIS_REST_URL") or not os.getenv(
        "UPSTASH_REDIS_REST_TOKEN"
    ):
        pytest.skip("Live Redis env vars are not available")

    adapter = RedisAdapter.from_env(
        os.environ["UPSTASH_REDIS_REST_URL"],
        os.environ["UPSTASH_REDIS_REST_TOKEN"],
    )

    key = "live:test:roundtrip"
    adapter.set(key, {"status": "ok"})
    assert adapter.get(key) == {"status": "ok"}
    adapter.delete(key)
