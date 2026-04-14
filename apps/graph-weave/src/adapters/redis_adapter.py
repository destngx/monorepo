"""
HTTP-based Redis client via Upstash REST API.

Implements core Redis operations using Upstash REST API with:
- Connection pooling via requests.Session
- Exponential backoff retry logic
- Bearer token authentication
- Type-safe operations with clear error handling
"""

import time
import json
import logging
from src.app_logging import get_logger
from typing import Any, Callable, Dict, List, Optional, TypeVar
from functools import wraps
import requests

T = TypeVar("T")

logger = get_logger(__name__)


class RedisError(Exception):
    """Base exception for Redis operations."""

    pass


class RedisTimeoutError(RedisError):
    """Raised when Redis operation times out."""

    pass


class RedisConnectionError(RedisError):
    """Raised when connection to Redis fails."""

    pass


def retry_with_backoff(
    max_retries: int = 3, initial_backoff_ms: int = 100
) -> Callable[[Callable[..., T]], Callable[..., T]]:
    def decorator(func: Callable[..., T]) -> Callable[..., T]:
        @wraps(func)
        def wrapper(*args: Any, **kwargs: Any) -> T:
            last_exception = None

            for attempt in range(max_retries):
                try:
                    return func(*args, **kwargs)
                except (
                    requests.Timeout,
                    requests.ConnectionError,
                    RedisTimeoutError,
                ) as e:
                    last_exception = e
                    if attempt < max_retries - 1:
                        backoff_ms = initial_backoff_ms * (2**attempt)
                        wait_s = backoff_ms / 1000.0
                        logger.warning(
                            f"Attempt {attempt + 1}/{max_retries} failed: {str(e)}, "
                            f"retrying in {backoff_ms}ms"
                        )
                        time.sleep(wait_s)
                    else:
                        logger.error(f"All {max_retries} attempts failed: {str(e)}")
                except Exception as e:
                    # Non-retryable errors (400, 403, malformed response, etc)
                    raise

            # All retries exhausted
            if isinstance(last_exception, (requests.Timeout, requests.ConnectionError)):
                raise RedisTimeoutError(
                    f"Redis operation timed out after {max_retries} retries"
                ) from last_exception
            if last_exception is not None:
                raise last_exception
            raise RedisError("Redis operation failed")

        return wrapper

    return decorator


class UpstashRedisClient:
    """HTTP-based Redis client for Upstash REST API."""

    url: str
    token: str
    _session: requests.Session

    def __init__(self, url: str, token: str) -> None:
        """
        Initialize Upstash Redis client.

        Args:
            url: Upstash REST API endpoint (e.g., https://region.upstash.io)
            token: Bearer token for authentication

        Raises:
            ValueError: If url or token is empty
        """
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
        """
        Execute Redis command via Upstash REST API.

        Args:
            command: List of command and arguments (e.g., ["SET", "key", "value"])

        Returns:
            Command result

        Raises:
            RedisError: On API errors
            RedisTimeoutError: On timeout
        """
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

    def _serialize(self, value: Any) -> Any:
        """Serialize complex types to JSON strings."""
        if isinstance(value, (str, int, float, bool)) or value is None:
            return value
        return json.dumps(value)

    def _deserialize(self, value: Any) -> Any:
        """Deserialize JSON strings to Python objects if possible."""
        if isinstance(value, str):
            # Check if it looks like JSON
            if (value.startswith("{") and value.endswith("}")) or (
                value.startswith("[") and value.endswith("]")
            ):
                try:
                    return json.loads(value)
                except json.JSONDecodeError:
                    return value
        return value

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def get(self, key: str) -> Optional[Any]:
        """
        Get value for key.

        Args:
            key: Redis key

        Returns:
            Value if key exists, None otherwise

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["GET", key])
            return self._deserialize(result)
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        """
        Set key to value.

        Args:
            key: Redis key
            value: Value to set (will be JSON serialized)
            ex: Optional expiration in seconds

        Returns:
            True if successful

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            cmd = ["SET", key, self._serialize(value)]
            if ex:
                cmd.extend(["EX", str(ex)])

            result = self._execute(cmd)
            return result == "OK"
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def delete(self, key: str) -> bool:
        """
        Delete key.

        Args:
            key: Redis key

        Returns:
            True if key was deleted, False if didn't exist

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["DEL", key])
            return result > 0 if result is not None else False
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def exists(self, key: str) -> bool:
        """
        Check if key exists.

        Args:
            key: Redis key

        Returns:
            True if key exists, False otherwise

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["EXISTS", key])
            return result > 0 if result is not None else False
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def ttl(self, key: str) -> int:
        """
        Get time to live for key in seconds.

        Args:
            key: Redis key

        Returns:
            Seconds remaining:
            - -1 if key exists but has no expiration
            - -2 if key does not exist
            - positive number otherwise

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["TTL", key])
            return result if result is not None else -2
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def rpush(self, key: str, value: Any) -> int:
        """
        Append value to list.

        Args:
            key: Redis key
            value: Value to append

        Returns:
            Length of list after push

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["RPUSH", key, self._serialize(value)])
            return result if result is not None else 0
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def lpush(self, key: str, value: Any) -> int:
        """
        Prepend value to list.

        Args:
            key: Redis key
            value: Value to prepend

        Returns:
            Length of list after push

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["LPUSH", key, self._serialize(value)])
            return result if result is not None else 0
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        """
        Get range of elements from list.

        Args:
            key: Redis key
            start: Start index (inclusive)
            end: End index (inclusive)

        Returns:
            List of values in range

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["LRANGE", key, start, end])
            items = result if result is not None else []
            return [self._deserialize(item) for item in items]
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def ltrim(self, key: str, start: int, end: int) -> bool:
        """
        Trim list to range.

        Args:
            key: Redis key
            start: Start index (inclusive)
            end: End index (inclusive)

        Returns:
            True if successful

        Raises:
            RedisError: On Redis errors
        """
        if not key:
            raise ValueError("Key cannot be empty")

        try:
            result = self._execute(["LTRIM", key, start, end])
            return result == "OK"
        except RedisError:
            raise

    @retry_with_backoff(max_retries=3, initial_backoff_ms=100)
    def mget(self, keys: List[str]) -> Dict[str, Any]:
        """
        Get multiple keys at once.

        Args:
            keys: List of Redis keys

        Returns:
            Dictionary mapping keys to values (missing keys omitted)

        Raises:
            RedisError: On Redis errors
        """
        if not keys:
            return {}

        try:
            result = self._execute(["MGET"] + keys)
            # Upstash returns array, we map back to dict
            values = result if result is not None else []
            return {
                key: self._deserialize(value)
                for key, value in zip(keys, values)
                if value is not None
            }
        except RedisError:
            raise

    def close(self):
        """Close the session."""
        self._session.close()
        logger.info("UpstashRedisClient session closed")
