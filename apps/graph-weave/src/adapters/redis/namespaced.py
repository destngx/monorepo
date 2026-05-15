import logging
import threading
from typing import Any, Dict, List, Optional
from .upstash import UpstashRedisClient
from .fallback import FallbackStorage
from .circuit_breaker import CircuitBreaker, CircuitBreakerState
from .exceptions import RedisError, RedisTimeoutError, RedisConnectionError

logger = logging.getLogger(__name__)

TTL_CONFIG = {
    "run": 7 * 24 * 3600,
    "workflow": None,
    "thread": 1 * 3600,
    "checkpoint": 30 * 24 * 3600,
    "event": None,
}

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
    def workflow_key(
        workflow_id: str, tenant_id: str, version: Optional[str] = None
    ) -> str:
        if version:
            return f"workflow:{tenant_id}:{workflow_id}:{version}"
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
        return self._execute_with_fallback(
            "GET", lambda: self.redis_client.get(key), lambda: self.fallback_storage.get(key)
        )

    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        ttl = ex or self._get_ttl(key)
        return self._execute_with_fallback(
            "SET",
            lambda: self.redis_client.set(key, value, ex=ttl),
            lambda: self.fallback_storage.set(key, value, ex=ttl),
        )

    def delete(self, key: str) -> bool:
        return self._execute_with_fallback(
            "DELETE", lambda: self.redis_client.delete(key), lambda: self.fallback_storage.delete(key)
        )

    def exists(self, key: str) -> bool:
        return self._execute_with_fallback(
            "EXISTS", lambda: self.redis_client.exists(key), lambda: self.fallback_storage.exists(key)
        )

    def rpush(self, key: str, value: Any) -> int:
        return self._execute_with_fallback(
            "RPUSH", lambda: self.redis_client.rpush(key, value), lambda: self.fallback_storage.rpush(key, value)
        )

    def lpush(self, key: str, value: Any) -> int:
        return self._execute_with_fallback(
            "LPUSH", lambda: self.redis_client.lpush(key, value), lambda: self.fallback_storage.lpush(key, value)
        )

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        return self._execute_with_fallback(
            "LRANGE",
            lambda: self.redis_client.lrange(key, start, end),
            lambda: self.fallback_storage.lrange(key, start, end),
        )

    def ltrim(self, key: str, start: int, end: int) -> bool:
        return self._execute_with_fallback(
            "LTRIM",
            lambda: self.redis_client.ltrim(key, start, end),
            lambda: self.fallback_storage.ltrim(key, start, end),
        )

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        return self._execute_with_fallback(
            "MGET", lambda: self.redis_client.mget(keys), lambda: self.fallback_storage.mget(keys)
        )

    def ttl(self, key: str) -> int:
        return self._execute_with_fallback(
            "TTL", lambda: self.redis_client.ttl(key), lambda: self.fallback_storage.ttl(key)
        )

    def hset(self, key: str, field: str, value: Any) -> int:
        return self._execute_with_fallback(
            "HSET",
            lambda: self.redis_client.hset(key, field, value),
            lambda: self.fallback_storage.hset(key, field, value),
        )

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        return self._execute_with_fallback(
            "HSETNX",
            lambda: self.redis_client.hsetnx(key, field, value),
            lambda: self.fallback_storage.hsetnx(key, field, value),
        )

    def hget(self, key: str, field: str) -> Optional[Any]:
        return self._execute_with_fallback(
            "HGET",
            lambda: self.redis_client.hget(key, field),
            lambda: self.fallback_storage.hget(key, field),
        )

    def hdel(self, key: str, field: str) -> int:
        return self._execute_with_fallback(
            "HDEL",
            lambda: self.redis_client.hdel(key, field),
            lambda: self.fallback_storage.hdel(key, field),
        )

    def hgetall(self, key: str) -> Dict[str, Any]:
        return self._execute_with_fallback(
            "HGETALL",
            lambda: self.redis_client.hgetall(key),
            lambda: self.fallback_storage.hgetall(key),
        )

    def sadd(self, key: str, *values: Any) -> int:
        return self._execute_with_fallback(
            "SADD",
            lambda: self.redis_client.sadd(key, *values),
            lambda: self.fallback_storage.sadd(key, *values) if hasattr(self.fallback_storage, 'sadd') else 0,
        )

    def smembers(self, key: str) -> set:
        return self._execute_with_fallback(
            "SMEMBERS",
            lambda: self.redis_client.smembers(key),
            lambda: self.fallback_storage.smembers(key) if hasattr(self.fallback_storage, 'smembers') else set(),
        )

    def srem(self, key: str, *values: Any) -> int:
        return self._execute_with_fallback(
            "SREM",
            lambda: self.redis_client.srem(key, *values),
            lambda: self.fallback_storage.srem(key, *values) if hasattr(self.fallback_storage, 'srem') else 0,
        )

    def sinter(self, *keys: str) -> set:
        return self._execute_with_fallback(
            "SINTER",
            lambda: self.redis_client.sinter(*keys),
            lambda: self.fallback_storage.sinter(*keys) if hasattr(self.fallback_storage, 'sinter') else set(),
        )

    def keys(self, pattern: str) -> List[str]:
        return self._execute_with_fallback(
            "KEYS", lambda: self.redis_client.keys(pattern), lambda: self.fallback_storage.keys(pattern)
        )

    def get_health(self) -> Dict[str, Any]:
        is_closed = self.circuit_breaker.state == CircuitBreakerState.CLOSED
        return {
            "status": "healthy" if is_closed else "degraded",
            "redis": is_closed,
            "circuit_breaker": self.circuit_breaker.get_state(),
        }

    def clear(self):
        """Clear all data from both Redis and fallback storage."""
        def redis_clear():
            if hasattr(self.redis_client, "clear"):
                self.redis_client.clear()
            return True
        return self._execute_with_fallback("CLEAR", redis_clear, lambda: self.fallback_storage.clear())

    def close(self):
        self.redis_client.close()
        self.fallback_storage.clear()
