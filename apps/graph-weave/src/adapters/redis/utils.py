import time
import json
import logging
import requests
from functools import wraps
from typing import Any, Callable, TypeVar
from .exceptions import RedisError, RedisTimeoutError

T = TypeVar("T")
logger = logging.getLogger(__name__)

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

def serialize(value: Any) -> Any:
    """Serialize complex types to JSON strings."""
    if isinstance(value, (str, int, float, bool)) or value is None:
        return value
    return json.dumps(value)

def deserialize(value: Any) -> Any:
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
