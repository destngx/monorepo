"""
E2E Integration tests for UpstashRedisClient with real Redis.
"""

import pytest
import os
import time
import threading
from concurrent.futures import ThreadPoolExecutor

from src.adapters.redis_adapter import (
    UpstashRedisClient,
    RedisError,
    RedisTimeoutError,
    RedisConnectionError,
)

# ============================================================================
# REAL REDIS INTEGRATION TESTS
# ============================================================================

class TestUpstashRedisClientIntegration:
    """Integration tests with real Upstash instance."""

    @pytest.fixture
    def client(self):
        """Create client for integration tests."""
        url = os.getenv("UPSTASH_REDIS_REST_URL")
        token = os.getenv("UPSTASH_REDIS_REST_TOKEN")
        
        if not url or not token:
            pytest.fail("UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN are required for e2e tests")
            
        client = UpstashRedisClient(url, token)
        yield client
        client.close()

    def test_set_get_roundtrip_string(self, client):
        """SET/GET roundtrip with string value."""
        key = "test:string:value"
        value = "hello world"

        # Clean up
        client.delete(key)

        # Set and get
        assert client.set(key, value) is True
        assert client.get(key) == value

        # Clean up
        client.delete(key)

    def test_set_get_roundtrip_json(self, client):
        """SET/GET roundtrip with JSON value."""
        key = "test:json:value"
        value = {"name": "Alice", "age": 30, "active": True}

        # Clean up
        client.delete(key)

        # Set and get
        assert client.set(key, value) is True
        result = client.get(key)
        assert result == value

        # Clean up
        client.delete(key)

    def test_delete_removes_key(self, client):
        """DELETE removes key successfully."""
        key = "test:delete:key"

        client.set(key, "value")
        assert client.exists(key) is True

        assert client.delete(key) is True
        assert client.exists(key) is False

    def test_get_on_deleted_key_returns_none(self, client):
        """GET on deleted key returns None."""
        key = "test:deleted:key"

        client.set(key, "value")
        client.delete(key)

        assert client.get(key) is None

    def test_exists_returns_true_false(self, client):
        """EXISTS returns True/False correctly."""
        key = "test:exists:key"

        client.delete(key)
        assert client.exists(key) is False

        client.set(key, "value")
        assert client.exists(key) is True

        client.delete(key)

    def test_ttl_returns_correct_seconds(self, client):
        """TTL returns correct remaining seconds."""
        key = "test:ttl:key"

        # Set with 60 second expiration
        client.delete(key)
        client.set(key, "value", ex=60)

        ttl = client.ttl(key)
        assert ttl > 0
        assert ttl <= 60

        # Clean up
        client.delete(key)

    def test_ttl_returns_minus_one_for_no_expiry(self, client):
        """TTL returns -1 for key with no expiration."""
        key = "test:ttl:no_expiry"

        client.delete(key)
        client.set(key, "value")  # No ex parameter

        ttl = client.ttl(key)
        assert ttl == -1

        client.delete(key)

    def test_ttl_returns_minus_two_for_missing_key(self, client):
        """TTL returns -2 for non-existent key."""
        key = "test:ttl:missing"
        client.delete(key)

        ttl = client.ttl(key)
        assert ttl == -2

    def test_rpush_appends_to_list(self, client):
        """RPUSH appends to list."""
        key = "test:list:rpush"

        client.delete(key)

        len1 = client.rpush(key, "first")
        assert len1 == 1

        len2 = client.rpush(key, "second")
        assert len2 == 2

        client.delete(key)

    def test_lrange_retrieves_list_items(self, client):
        """LRANGE retrieves all list items in order."""
        key = "test:list:lrange"

        client.delete(key)
        client.rpush(key, "a")
        client.rpush(key, "b")
        client.rpush(key, "c")

        items = client.lrange(key, 0, -1)
        assert items == ["a", "b", "c"]

        client.delete(key)

    def test_ltrim_truncates_list(self, client):
        """LTRIM truncates list correctly."""
        key = "test:list:ltrim"

        client.delete(key)
        client.rpush(key, "a")
        client.rpush(key, "b")
        client.rpush(key, "c")
        client.rpush(key, "d")

        assert client.ltrim(key, 1, 2) is True

        items = client.lrange(key, 0, -1)
        assert items == ["b", "c"]

        client.delete(key)

    def test_mget_retrieves_multiple_keys(self, client):
        """MGET retrieves multiple keys."""
        key1 = "test:mget:1"
        key2 = "test:mget:2"
        key3 = "test:mget:3"

        client.delete(key1)
        client.delete(key2)
        client.delete(key3)

        client.set(key1, "value1")
        client.set(key2, "value2")
        # key3 not set

        result = client.mget([key1, key2, key3])
        assert result == {key1: "value1", key2: "value2"}

        client.delete(key1)
        client.delete(key2)

    def test_mget_empty_list(self, client):
        """MGET with empty list returns empty dict."""
        client = UpstashRedisClient(os.getenv("UPSTASH_REDIS_REST_URL"), os.getenv("UPSTASH_REDIS_REST_TOKEN"))
        result = client.mget([])
        assert result == {}
        client.close()


class TestUpstashRedisClientConcurrency:
    """Test concurrent operations."""

    @pytest.fixture
    def client(self):
        """Create client for concurrency tests."""
        url = os.getenv("UPSTASH_REDIS_REST_URL")
        token = os.getenv("UPSTASH_REDIS_REST_TOKEN")
        client = UpstashRedisClient(url, token)
        yield client
        client.close()

    def test_concurrent_set_operations(self, client):
        """Multiple threads can SET concurrently."""

        def set_value(thread_id):
            key = f"test:concurrent:set:{thread_id}"
            client.delete(key)
            client.set(key, f"value_{thread_id}")
            return True

        with ThreadPoolExecutor(max_workers=5) as executor:
            results = list(executor.map(set_value, range(10)))

        assert all(results)

        # Cleanup
        for i in range(10):
            client.delete(f"test:concurrent:set:{i}")

    def test_concurrent_get_operations(self, client):
        """Multiple threads can GET concurrently."""
        key = "test:concurrent:get:shared"
        client.set(key, "shared_value")

        def get_value(_):
            return client.get(key) == "shared_value"

        with ThreadPoolExecutor(max_workers=5) as executor:
            results = list(executor.map(get_value, range(10)))

        assert all(results)

        # Cleanup
        client.delete(key)

    def test_concurrent_rpush_increments_length(self, client):
        """Concurrent RPUSH operations increment list length."""
        key = "test:concurrent:rpush"
        client.delete(key)

        def push_value(i):
            return client.rpush(key, f"item_{i}")

        with ThreadPoolExecutor(max_workers=5) as executor:
            results = list(executor.map(push_value, range(10)))

        # All pushes should succeed, last one should return 10
        assert max(results) == 10

        items = client.lrange(key, 0, -1)
        assert len(items) == 10

        client.delete(key)
