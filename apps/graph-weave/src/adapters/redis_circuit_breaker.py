"""
Redis circuit breaker with tenant-scoped namespacing and fallback storage.

Implements:
- Tenant-scoped key namespacing (run, workflow, thread, checkpoint, event)
- Circuit breaker pattern (CLOSED → OPEN → HALF_OPEN)
- Fallback in-memory storage with LRU eviction
- Health check endpoint support
"""

import time
import logging
import threading
from enum import Enum
from collections import OrderedDict
from typing import Any, Dict, List, Optional
from dataclasses import dataclass

from .redis_adapter import (
    UpstashRedisClient,
    RedisError,
    RedisTimeoutError,
    RedisConnectionError,
)

logger = logging.getLogger(__name__)

TTL_CONFIG = {
    "run": 7 * 24 * 3600,
    "workflow": None,
    "thread": 1 * 3600,
    "checkpoint": 30 * 24 * 3600,
    "event": None,
}


class CircuitBreakerState(Enum):
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"


class FallbackStorage:
    """In-memory LRU storage for when Redis is unavailable."""

    MAX_KEYS = 1000

    def __init__(self):
        self._store: OrderedDict[str, Any] = OrderedDict()
        self._lock = threading.Lock()

    def get(self, key: str) -> Optional[Any]:
        with self._lock:
            if key in self._store:
                self._store.move_to_end(key)
                return self._store[key]
            return None

    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        with self._lock:
            if len(self._store) >= self.MAX_KEYS:
                oldest = next(iter(self._store))
                del self._store[oldest]

            self._store[key] = value
            if key in self._store:
                self._store.move_to_end(key)
            return True

    def delete(self, key: str) -> bool:
        with self._lock:
            if key in self._store:
                del self._store[key]
                return True
            return False

    def exists(self, key: str) -> bool:
        with self._lock:
            return key in self._store

    def rpush(self, key: str, value: Any) -> int:
        with self._lock:
            if key not in self._store:
                self._store[key] = []
            elif not isinstance(self._store[key], list):
                self._store[key] = [self._store[key]]

            self._store[key].append(value)
            return len(self._store[key])

    def lpush(self, key: str, value: Any) -> int:
        with self._lock:
            if key not in self._store:
                self._store[key] = []
            elif not isinstance(self._store[key], list):
                self._store[key] = [self._store[key]]

            self._store[key].insert(0, value)
            return len(self._store[key])

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], list):
                return []
            items = self._store[key]
            if end == -1:
                return items[start:]
            return items[start : end + 1]

    def ltrim(self, key: str, start: int, end: int) -> bool:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], list):
                return False
            if end == -1:
                self._store[key] = self._store[key][start:]
            else:
                self._store[key] = self._store[key][start : end + 1]
            return True

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        with self._lock:
            return {k: self._store[k] for k in keys if k in self._store}

    def ttl(self, key: str) -> int:
        with self._lock:
            if key not in self._store:
                return -2
            return -1

    def clear(self):
        with self._lock:
            self._store.clear()

    def hset(self, key: str, field: str, value: Any) -> int:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], dict):
                self._store[key] = {}
            self._store[key][field] = value
            return 1

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], dict):
                self._store[key] = {}
            if field in self._store[key]:
                return 0
            self._store[key][field] = value
            return 1

    def hget(self, key: str, field: str) -> Optional[Any]:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                return self._store[key].get(field)
            return None

    def hdel(self, key: str, field: str) -> int:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                if field in self._store[key]:
                    del self._store[key][field]
                    return 1
            return 0

    def hgetall(self, key: str) -> Dict[str, Any]:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                return dict(self._store[key])
            return {}


class CircuitBreaker:
    """Circuit breaker for Redis availability."""

    def __init__(self, failure_threshold: int = 3, backoff_ms: int = 100):
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.failure_threshold = failure_threshold
        self.backoff_ms = backoff_ms
        self.last_failure_time: Optional[float] = None
        self._lock = threading.Lock()

    def record_success(self):
        with self._lock:
            if self.state == CircuitBreakerState.HALF_OPEN:
                logger.info("Circuit breaker: HALF_OPEN → CLOSED (success)")
                self.state = CircuitBreakerState.CLOSED
                self.failure_count = 0
            elif self.state == CircuitBreakerState.CLOSED:
                self.failure_count = 0

    def record_failure(self):
        with self._lock:
            self.failure_count += 1
            self.last_failure_time = time.monotonic()

            if self.state == CircuitBreakerState.HALF_OPEN:
                logger.warning(
                    f"Circuit breaker: HALF_OPEN → OPEN (failure {self.failure_count})"
                )
                self.state = CircuitBreakerState.OPEN
            elif (
                self.failure_count >= self.failure_threshold
                and self.state == CircuitBreakerState.CLOSED
            ):
                logger.warning(
                    f"Circuit breaker: CLOSED → OPEN ({self.failure_count} failures)"
                )
                self.state = CircuitBreakerState.OPEN

    def can_attempt(self) -> bool:
        with self._lock:
            if self.state == CircuitBreakerState.CLOSED:
                return True

            if self.state == CircuitBreakerState.OPEN:
                if self.last_failure_time is None:
                    return False

                elapsed_ms = (time.monotonic() - self.last_failure_time) * 1000
                if elapsed_ms >= self.backoff_ms:
                    logger.info("Circuit breaker: OPEN → HALF_OPEN (backoff expired)")
                    self.state = CircuitBreakerState.HALF_OPEN
                    return True
                return False

            if self.state == CircuitBreakerState.HALF_OPEN:
                return True

            return False

    def get_state(self) -> str:
        with self._lock:
            return self.state.value


class NamespacedRedisClient:
    """Redis client with tenant-scoped namespacing and circuit breaker."""

    def __init__(
        self,
        redis_client: UpstashRedisClient,
        fallback_storage: Optional[FallbackStorage] = None,
    ):
        self.redis_client = redis_client
        self.fallback_storage = fallback_storage or FallbackStorage()
        self.circuit_breaker = CircuitBreaker()
        self._lock = threading.Lock()

    @staticmethod
    def run_key(run_id: str, tenant_id: str) -> str:
        return f"run:{tenant_id}:{run_id}"

    @staticmethod
    def workflow_key(workflow_id: str, tenant_id: str) -> str:
        return f"workflow:{tenant_id}:{workflow_id}"

    @staticmethod
    def thread_key(thread_id: str, tenant_id: str) -> str:
        return f"thread:{tenant_id}:{thread_id}"

    @staticmethod
    def checkpoint_key(checkpoint_id: str, tenant_id: str) -> str:
        return f"checkpoint:{tenant_id}:{checkpoint_id}"

    @staticmethod
    def event_key(event_id: str, tenant_id: str) -> str:
        return f"event:{tenant_id}:{event_id}"

    @staticmethod
    def schedule_key(tenant_id: str) -> str:
        return f"schedules:{tenant_id}"

    def _get_ttl(self, key: str) -> Optional[int]:
        for key_type, ttl in TTL_CONFIG.items():
            if key.startswith(f"{key_type}:"):
                return ttl
        return None

    def _execute_with_fallback(self, operation_name: str, redis_op, fallback_op) -> Any:
        if not self.circuit_breaker.can_attempt():
            logger.debug(f"Circuit breaker OPEN: using fallback for {operation_name}")
            try:
                return fallback_op()
            except Exception as e:
                logger.error(f"Fallback operation failed: {e}")
                raise

        try:
            result = redis_op()
            self.circuit_breaker.record_success()
            return result
        except (RedisError, RedisTimeoutError, RedisConnectionError) as e:
            logger.warning(f"Redis operation failed: {e}")
            self.circuit_breaker.record_failure()
            return fallback_op()

    def get(self, key: str) -> Optional[Any]:
        def redis_get():
            return self.redis_client.get(key)

        def fallback_get():
            return self.fallback_storage.get(key)

        return self._execute_with_fallback("GET", redis_get, fallback_get)

    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        ttl = ex or self._get_ttl(key)

        def redis_set():
            return self.redis_client.set(key, value, ex=ttl)

        def fallback_set():
            return self.fallback_storage.set(key, value, ex=ttl)

        return self._execute_with_fallback("SET", redis_set, fallback_set)

    def delete(self, key: str) -> bool:
        def redis_delete():
            return self.redis_client.delete(key)

        def fallback_delete():
            return self.fallback_storage.delete(key)

        return self._execute_with_fallback("DELETE", redis_delete, fallback_delete)

    def exists(self, key: str) -> bool:
        def redis_exists():
            return self.redis_client.exists(key)

        def fallback_exists():
            return self.fallback_storage.exists(key)

        return self._execute_with_fallback("EXISTS", redis_exists, fallback_exists)

    def rpush(self, key: str, value: Any) -> int:
        def redis_rpush():
            return self.redis_client.rpush(key, value)

        def fallback_rpush():
            return self.fallback_storage.rpush(key, value)

        return self._execute_with_fallback("RPUSH", redis_rpush, fallback_rpush)

    def lpush(self, key: str, value: Any) -> int:
        def redis_lpush():
            return self.redis_client.lpush(key, value)

        def fallback_lpush():
            return self.fallback_storage.lpush(key, value)

        return self._execute_with_fallback("LPUSH", redis_lpush, fallback_lpush)

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        def redis_lrange():
            return self.redis_client.lrange(key, start, end)

        def fallback_lrange():
            return self.fallback_storage.lrange(key, start, end)

        return self._execute_with_fallback("LRANGE", redis_lrange, fallback_lrange)

    def ltrim(self, key: str, start: int, end: int) -> bool:
        def redis_ltrim():
            return self.redis_client.ltrim(key, start, end)

        def fallback_ltrim():
            return self.fallback_storage.ltrim(key, start, end)

        return self._execute_with_fallback("LTRIM", redis_ltrim, fallback_ltrim)

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        def redis_mget():
            return self.redis_client.mget(keys)

        def fallback_mget():
            return self.fallback_storage.mget(keys)

        return self._execute_with_fallback("MGET", redis_mget, fallback_mget)

    def ttl(self, key: str) -> int:
        def redis_ttl():
            return self.redis_client.ttl(key)

        def fallback_ttl():
            return self.fallback_storage.ttl(key)

        return self._execute_with_fallback("TTL", redis_ttl, fallback_ttl)

    def hset(self, key: str, field: str, value: Any) -> int:
        def redis_hset():
            return self.redis_client.hset(key, field, value)

        def fallback_hset():
            return self.fallback_storage.hset(key, field, value)

        return self._execute_with_fallback("HSET", redis_hset, fallback_hset)

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        def redis_hsetnx():
            return self.redis_client.hsetnx(key, field, value)

        def fallback_hsetnx():
            return self.fallback_storage.hsetnx(key, field, value)

        return self._execute_with_fallback("HSETNX", redis_hsetnx, fallback_hsetnx)

    def hget(self, key: str, field: str) -> Optional[Any]:
        def redis_hget():
            return self.redis_client.hget(key, field)

        def fallback_hget():
            return self.fallback_storage.hget(key, field)

        return self._execute_with_fallback("HGET", redis_hget, fallback_hget)

    def hdel(self, key: str, field: str) -> int:
        def redis_hdel():
            return self.redis_client.hdel(key, field)

        def fallback_hdel():
            return self.fallback_storage.hdel(key, field)

        return self._execute_with_fallback("HDEL", redis_hdel, fallback_hdel)

    def hgetall(self, key: str) -> Dict[str, Any]:
        def redis_hgetall():
            return self.redis_client.hgetall(key)

        def fallback_hgetall():
            return self.fallback_storage.hgetall(key)

        return self._execute_with_fallback("HGETALL", redis_hgetall, fallback_hgetall)

    def get_health(self) -> Dict[str, Any]:
        is_closed = self.circuit_breaker.state == CircuitBreakerState.CLOSED

        return {
            "status": "healthy" if is_closed else "degraded",
            "redis": is_closed,
            "circuit_breaker": self.circuit_breaker.get_state(),
        }

    def clear(self):
        """Clear all data from both Redis and fallback storage (for testing)."""
        def redis_clear():
            if hasattr(self.redis_client, "clear"):
                self.redis_client.clear()
            return True

        def fallback_clear():
            self.fallback_storage.clear()
            return True

        return self._execute_with_fallback("CLEAR", redis_clear, fallback_clear)

    def close(self):
        self.redis_client.close()
        self.fallback_storage.clear()
