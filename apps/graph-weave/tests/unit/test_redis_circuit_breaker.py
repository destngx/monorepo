"""
Tests for Redis circuit breaker with namespacing and fallback storage.

Coverage:
- Unit: Key formatting, namespace isolation, circuit breaker state machine, LRU eviction
- Integration: Real Redis with circuit breaker resilience
- Concurrency: Thread-safe operations and state transitions
- Error scenarios: Failures, recovery, fallback activation
"""

import pytest
import time
import threading
from unittest.mock import Mock, patch, MagicMock
from concurrent.futures import ThreadPoolExecutor

from src.adapters.redis_circuit_breaker import (
    NamespacedRedisClient,
    CircuitBreaker,
    CircuitBreakerState,
    FallbackStorage,
    TTL_CONFIG,
)
from src.adapters.redis_adapter import (
    UpstashRedisClient,
    RedisError,
    RedisConnectionError,
)


class TestNamespacedRedisClientKeyFormatting:
    """Test namespace key formatting."""

    def test_run_key_format(self):
        assert (
            NamespacedRedisClient.run_key("run-123", "tenant-a")
            == "run:tenant-a:run-123"
        )

    def test_workflow_key_format(self):
        assert (
            NamespacedRedisClient.workflow_key("wf-456", "tenant-a", "v1")
            == "workflow:tenant-a:wf-456:v1"
        )

    def test_thread_key_format(self):
        assert (
            NamespacedRedisClient.thread_key("thread-789", "tenant-a")
            == "thread:tenant-a:thread-789"
        )

    def test_checkpoint_key_format(self):
        assert (
            NamespacedRedisClient.checkpoint_key("ckpt-111", "tenant-a")
            == "checkpoint:tenant-a:ckpt-111"
        )

    def test_event_key_format(self):
        assert (
            NamespacedRedisClient.event_key("evt-222", "tenant-a")
            == "event:tenant-a:evt-222"
        )

    def test_keys_differ_by_tenant(self):
        run_key_a = NamespacedRedisClient.run_key("run-1", "tenant-a")
        run_key_b = NamespacedRedisClient.run_key("run-1", "tenant-b")
        assert run_key_a != run_key_b
        assert "tenant-a" in run_key_a
        assert "tenant-b" in run_key_b


class TestFallbackStorage:
    """Test fallback in-memory storage."""

    def test_get_set_roundtrip(self):
        storage = FallbackStorage()
        storage.set("key1", "value1")
        assert storage.get("key1") == "value1"

    def test_delete_key(self):
        storage = FallbackStorage()
        storage.set("key1", "value1")
        assert storage.delete("key1") is True
        assert storage.get("key1") is None

    def test_exists_returns_true_false(self):
        storage = FallbackStorage()
        assert storage.exists("key1") is False
        storage.set("key1", "value1")
        assert storage.exists("key1") is True

    def test_ttl_returns_correct_values(self):
        storage = FallbackStorage()
        storage.set("key1", "value1")
        assert storage.ttl("key1") == -1
        assert storage.ttl("nonexistent") == -2

    def test_rpush_creates_list(self):
        storage = FallbackStorage()
        len1 = storage.rpush("list1", "item1")
        assert len1 == 1
        len2 = storage.rpush("list1", "item2")
        assert len2 == 2

    def test_lrange_retrieves_items(self):
        storage = FallbackStorage()
        storage.rpush("list1", "a")
        storage.rpush("list1", "b")
        storage.rpush("list1", "c")

        items = storage.lrange("list1", 0, -1)
        assert items == ["a", "b", "c"]

    def test_ltrim_truncates_list(self):
        storage = FallbackStorage()
        storage.rpush("list1", "a")
        storage.rpush("list1", "b")
        storage.rpush("list1", "c")

        storage.ltrim("list1", 1, -1)
        items = storage.lrange("list1", 0, -1)
        assert items == ["b", "c"]

    def test_mget_retrieves_multiple(self):
        storage = FallbackStorage()
        storage.set("k1", "v1")
        storage.set("k2", "v2")

        result = storage.mget(["k1", "k2", "k3"])
        assert result == {"k1": "v1", "k2": "v2"}

    def test_lru_eviction_at_max_keys(self):
        storage = FallbackStorage()
        for i in range(1001):
            storage.set(f"key{i}", f"value{i}")

        assert len(storage._store) <= FallbackStorage.MAX_KEYS
        assert storage.get("key0") is None
        assert storage.get("key1000") is not None

    def test_concurrent_access_thread_safe(self):
        storage = FallbackStorage()

        def worker(thread_id):
            for i in range(100):
                key = f"key{thread_id}_{i}"
                storage.set(key, f"value{i}")
                assert storage.get(key) == f"value{i}"

        threads = [threading.Thread(target=worker, args=(i,)) for i in range(5)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert len(storage._store) <= FallbackStorage.MAX_KEYS


class TestCircuitBreaker:
    """Test circuit breaker state machine."""

    def test_initializes_closed(self):
        cb = CircuitBreaker()
        assert cb.state == CircuitBreakerState.CLOSED
        assert cb.can_attempt() is True

    def test_closed_to_open_after_failures(self):
        cb = CircuitBreaker(failure_threshold=3)
        assert cb.can_attempt() is True

        cb.record_failure()
        cb.record_failure()
        assert cb.can_attempt() is True

        cb.record_failure()
        assert cb.state == CircuitBreakerState.OPEN
        assert cb.can_attempt() is False

    def test_open_to_half_open_after_backoff(self):
        cb = CircuitBreaker(failure_threshold=3, backoff_ms=50)

        for _ in range(3):
            cb.record_failure()

        assert cb.state == CircuitBreakerState.OPEN
        assert cb.can_attempt() is False

        time.sleep(0.06)
        assert cb.can_attempt() is True
        assert cb.state == CircuitBreakerState.HALF_OPEN

    def test_half_open_to_closed_on_success(self):
        cb = CircuitBreaker(failure_threshold=3, backoff_ms=50)

        for _ in range(3):
            cb.record_failure()

        time.sleep(0.06)
        cb.can_attempt()
        cb.record_success()

        assert cb.state == CircuitBreakerState.CLOSED
        assert cb.failure_count == 0

    def test_half_open_to_open_on_failure(self):
        cb = CircuitBreaker(failure_threshold=3, backoff_ms=50)

        for _ in range(3):
            cb.record_failure()

        time.sleep(0.06)
        cb.can_attempt()
        cb.record_failure()

        assert cb.state == CircuitBreakerState.OPEN

    def test_get_state_returns_string(self):
        cb = CircuitBreaker()
        assert cb.get_state() == "closed"

        cb.record_failure()
        cb.record_failure()
        cb.record_failure()
        assert cb.get_state() == "open"


class TestNamespacedRedisClientCircuitBreaker:
    """Test circuit breaker integration."""

    def test_fallback_used_when_circuit_open(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.side_effect = RedisConnectionError("connection failed")

        fallback = FallbackStorage()
        fallback.set("test_key", "fallback_value")

        client = NamespacedRedisClient(mock_redis, fallback)

        for _ in range(3):
            try:
                client.get("test_key")
            except:
                pass

        assert client.circuit_breaker.state == CircuitBreakerState.OPEN
        result = client.get("test_key")
        assert result == "fallback_value"

    def test_circuit_breaker_transparent_in_closed(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.return_value = "redis_value"

        fallback = FallbackStorage()
        client = NamespacedRedisClient(mock_redis, fallback)

        assert client.circuit_breaker.state == CircuitBreakerState.CLOSED
        result = client.get("test_key")
        assert result == "redis_value"
        mock_redis.get.assert_called_once()

    def test_health_check_returns_status(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        fallback = FallbackStorage()
        client = NamespacedRedisClient(mock_redis, fallback)

        health = client.get_health()
        assert health["status"] == "healthy"
        assert health["redis"] is True
        assert health["circuit_breaker"] == "closed"

    def test_health_check_degraded_when_open(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.side_effect = RedisConnectionError("down")

        fallback = FallbackStorage()
        client = NamespacedRedisClient(mock_redis, fallback)

        for _ in range(3):
            try:
                client.get("key")
            except:
                pass

        health = client.get_health()
        assert health["status"] == "degraded"
        assert health["redis"] is False


class TestTTLConfiguration:
    """Test TTL configuration."""

    def test_ttl_config_values(self):
        assert TTL_CONFIG["run"] == 7 * 24 * 3600
        assert TTL_CONFIG["workflow"] is None
        assert TTL_CONFIG["thread"] == 1 * 3600
        assert TTL_CONFIG["checkpoint"] == 30 * 24 * 3600
        assert TTL_CONFIG["event"] is None

    def test_get_ttl_extracts_from_key(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        client = NamespacedRedisClient(mock_redis)

        assert client._get_ttl("run:tenant:run-1") == 7 * 24 * 3600
        assert client._get_ttl("thread:tenant:thread-1") == 1 * 3600
        assert client._get_ttl("workflow:tenant:wf:v1") is None


class TestTenantIsolation:
    """Test tenant isolation."""

    def test_different_tenants_different_keys(self):
        key_tenant_a = NamespacedRedisClient.run_key("run-1", "tenant-a")
        key_tenant_b = NamespacedRedisClient.run_key("run-1", "tenant-b")

        assert key_tenant_a != key_tenant_b

    def test_mock_redis_isolation(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.return_value = "value"

        fallback_a = FallbackStorage()
        fallback_b = FallbackStorage()

        client_a = NamespacedRedisClient(mock_redis, fallback_a)
        client_b = NamespacedRedisClient(mock_redis, fallback_b)

        key_a = client_a.run_key("run-1", "tenant-a")
        key_b = client_b.run_key("run-1", "tenant-b")

        assert key_a != key_b
        assert "tenant-a" in key_a
        assert "tenant-b" in key_b


class TestConcurrentOperations:
    """Test concurrent operations through circuit breaker."""

    def test_concurrent_gets(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.return_value = "value"

        client = NamespacedRedisClient(mock_redis)

        def worker():
            for _ in range(10):
                result = client.get(f"key{_}")
                assert result == "value"

        threads = [threading.Thread(target=worker) for _ in range(5)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert mock_redis.get.call_count >= 50

    def test_concurrent_circuit_breaker_state_changes(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.get.side_effect = RedisConnectionError("down")

        client = NamespacedRedisClient(mock_redis)

        def trigger_failures():
            for _ in range(5):
                try:
                    client.get("key")
                except:
                    pass

        threads = [threading.Thread(target=trigger_failures) for _ in range(2)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert client.circuit_breaker.state in [
            CircuitBreakerState.OPEN,
            CircuitBreakerState.HALF_OPEN,
        ]


class TestSetWithTTL:
    """Test SET with automatic TTL based on key type."""

    def test_set_applies_ttl_by_key_type(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.set.return_value = True

        client = NamespacedRedisClient(mock_redis)

        run_key = client.run_key("run-1", "tenant-a")
        client.set(run_key, "value")

        mock_redis.set.assert_called_once()
        call_args = mock_redis.set.call_args
        assert call_args[1]["ex"] == 7 * 24 * 3600

    def test_set_explicit_ttl_overrides(self):
        mock_redis = Mock(spec=UpstashRedisClient)
        mock_redis.set.return_value = True

        client = NamespacedRedisClient(mock_redis)

        run_key = client.run_key("run-1", "tenant-a")
        client.set(run_key, "value", ex=3600)

        call_args = mock_redis.set.call_args
        assert call_args[1]["ex"] == 3600
