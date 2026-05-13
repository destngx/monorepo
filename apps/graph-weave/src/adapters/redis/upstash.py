import json
import requests
from typing import Any, Dict, List, Optional
from src.app_logging import get_logger
from .exceptions import RedisError, RedisTimeoutError, RedisConnectionError
from .utils import retry_with_backoff, serialize, deserialize

logger = get_logger(__name__)

class UpstashRedisClient:
    """HTTP-based Redis client for Upstash REST API."""

    url: str
    token: str
    _session: requests.Session

    def __init__(self, url: str, token: str) -> None:
        """Initialize Upstash Redis client."""
        if not url:
            raise ValueError("Redis URL cannot be empty")
        if not token:
            raise ValueError("Redis token cannot be empty")

        self.url = url.rstrip("/")
        self.token = token
        self._session = requests.Session()
        self._session.headers.update(
            {
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
            }
        )
        logger.info("UpstashRedisClient initialized")

    def _execute(self, command: List[Any]) -> Any:
        """Execute Redis command via Upstash REST API."""
        try:
            response = self._session.post(self.url, json=command, timeout=5)

            # Check for HTTP errors
            if response.status_code >= 500:
                # 5xx errors are retryable
                raise requests.ConnectionError(f"Server error {response.status_code}")
            elif response.status_code >= 400:
                # 4xx errors are not retryable
                try:
                    error_detail = response.json()
                    error_msg = error_detail.get("error", response.text)
                except:
                    error_msg = response.text
                raise RedisError(f"Redis API error {response.status_code}: {error_msg}")

            # Parse successful response
            try:
                result = response.json()
                return result.get("result")
            except json.JSONDecodeError:
                raise RedisError(f"Invalid JSON response: {response.text}")

        except requests.Timeout:
            raise RedisTimeoutError("Request timed out")
        except requests.ConnectionError as e:
            raise RedisConnectionError(f"Connection failed: {str(e)}")

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def get(self, key: str) -> Optional[Any]:
        if not key:
            raise ValueError("Key cannot be empty")
        return deserialize(self._execute(["GET", key]))

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        if not key:
            raise ValueError("Key cannot be empty")
        cmd = ["SET", key, serialize(value)]
        if ex:
            cmd.extend(["EX", str(ex)])
        result = self._execute(cmd)
        return result == "OK"

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def delete(self, key: str) -> bool:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["DEL", key])
        return result > 0 if result is not None else False

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def exists(self, key: str) -> bool:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["EXISTS", key])
        return result > 0 if result is not None else False

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def ttl(self, key: str) -> int:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["TTL", key])
        return result if result is not None else -2

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def rpush(self, key: str, value: Any) -> int:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["RPUSH", key, serialize(value)])
        return result if result is not None else 0

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def lpush(self, key: str, value: Any) -> int:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["LPUSH", key, serialize(value)])
        return result if result is not None else 0

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["LRANGE", key, start, end])
        items = result if result is not None else []
        return [deserialize(item) for item in items]

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def ltrim(self, key: str, start: int, end: int) -> bool:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["LTRIM", key, start, end])
        return result == "OK"

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def mget(self, keys: List[str]) -> Dict[str, Any]:
        if not keys:
            return {}
        result = self._execute(["MGET"] + keys)
        values = result if result is not None else []
        return {
            key: deserialize(value)
            for key, value in zip(keys, values)
            if value is not None
        }

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def hset(self, key: str, field: str, value: Any) -> int:
        if not key or not field:
            raise ValueError("Key and field cannot be empty")
        result = self._execute(["HSET", key, field, serialize(value)])
        return result if result is not None else 0

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def hsetnx(self, key: str, field: str, value: Any) -> int:
        if not key or not field:
            raise ValueError("Key and field cannot be empty")
        result = self._execute(["HSETNX", key, field, serialize(value)])
        return result if result is not None else 0

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def hget(self, key: str, field: str) -> Optional[Any]:
        if not key or not field:
            raise ValueError("Key and field cannot be empty")
        result = self._execute(["HGET", key, field])
        return deserialize(result)

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def hgetall(self, key: str) -> Dict[str, Any]:
        if not key:
            raise ValueError("Key cannot be empty")
        result = self._execute(["HGETALL", key])
        if not result:
            return {}
        res_dict = {}
        for i in range(0, len(result), 2):
            field = result[i]
            value = result[i+1]
            res_dict[field] = deserialize(value)
        return res_dict

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def hdel(self, key: str, field: str) -> int:
        if not key or not field:
            raise ValueError("Key and field cannot be empty")
        result = self._execute(["HDEL", key, field])
        return result if result is not None else 0

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def keys(self, pattern: str) -> List[str]:
        result = self._execute(["KEYS", pattern])
        return result if result is not None else []

    def close(self):
        self._session.close()
        logger.info("UpstashRedisClient session closed")
