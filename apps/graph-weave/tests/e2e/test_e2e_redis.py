import os

import pytest

from src.adapters.cache import RedisAdapter


def test_live_redis_roundtrip():
    url = os.environ["UPSTASH_REDIS_REST_URL"]
    token = os.environ["UPSTASH_REDIS_REST_TOKEN"]

    adapter = RedisAdapter.from_env(url, token)

    key = "live:test:roundtrip"
    adapter.set(key, {"status": "ok"})
    assert adapter.get(key) == {"status": "ok"}
    adapter.delete(key)
