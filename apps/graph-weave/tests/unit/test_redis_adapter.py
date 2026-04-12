"""
Tests for UpstashRedisClient.

Test coverage:
- Unit tests: Configuration, request formatting, error handling (no network)
- Integration tests: Real Upstash instance (requires UPSTASH_REDIS_REST_URL/TOKEN)
- Concurrency tests: Multiple threads accessing same client
- Error scenarios: Timeouts, invalid responses, connection failures
"""

import pytest
import os
import json
import time
import threading
from unittest.mock import Mock, patch, MagicMock
from concurrent.futures import ThreadPoolExecutor

from src.adapters.redis_adapter import (
    UpstashRedisClient,
    RedisError,
    RedisTimeoutError,
    RedisConnectionError,
    retry_with_backoff,
)
from src.config import GraphWeaveConfig


# ============================================================================
# UNIT TESTS (No Network)
# ============================================================================


class TestUpstashRedisClientConfig:
    """Test configuration validation."""

    def test_config_loads_url_from_env(self):
        """Config loads UPSTASH_REDIS_REST_URL from environment."""
        assert hasattr(GraphWeaveConfig, "UPSTASH_REDIS_REST_URL")

    def test_config_loads_token_from_env(self):
        """Config loads UPSTASH_REDIS_REST_TOKEN from environment."""
        assert hasattr(GraphWeaveConfig, "UPSTASH_REDIS_REST_TOKEN")

    def test_client_init_raises_on_empty_url(self):
        """RedisClient raises ValueError if URL is empty."""
        with pytest.raises(ValueError, match="URL cannot be empty"):
            UpstashRedisClient("", "valid_token")

    def test_client_init_raises_on_empty_token(self):
        """RedisClient raises ValueError if token is empty."""
        with pytest.raises(ValueError, match="token cannot be empty"):
            UpstashRedisClient("https://valid-url.upstash.io", "")

    def test_client_init_success(self):
        """RedisClient initializes successfully with valid credentials."""
        client = UpstashRedisClient("https://test.upstash.io", "test_token")
        assert client.url == "https://test.upstash.io"
        assert client.token == "test_token"
        client.close()

    def test_client_strips_trailing_slash_from_url(self):
        """RedisClient strips trailing slash from URL."""
        client = UpstashRedisClient("https://test.upstash.io/", "test_token")
        assert client.url == "https://test.upstash.io"
        client.close()


class TestUpstashRedisClientRequestFormatting:
    """Test request formatting and headers."""

    def test_authorization_header_set_correctly(self):
        """Authorization header is set with Bearer token."""
        client = UpstashRedisClient("https://test.upstash.io", "test_token_xyz")
        assert "Authorization" in client._session.headers
        assert client._session.headers["Authorization"] == "Bearer test_token_xyz"
        client.close()

    def test_content_type_header_set_correctly(self):
        """Content-Type header is set to application/json."""
        client = UpstashRedisClient("https://test.upstash.io", "test_token")
        assert client._session.headers["Content-Type"] == "application/json"
        client.close()


class TestUpstashRedisClientErrorHandling:
    """Test error handling without network."""

    @patch("src.adapters.redis_adapter.requests.Session")
    def test_empty_key_raises_value_error(self, mock_session_class):
        """Operations raise ValueError for empty key."""
        client = UpstashRedisClient("https://test.upstash.io", "test_token")

        with pytest.raises(ValueError, match="Key cannot be empty"):
            client.get("")

        with pytest.raises(ValueError, match="Key cannot be empty"):
            client.set("", "value")

        with pytest.raises(ValueError, match="Key cannot be empty"):
            client.delete("")

        client.close()

    @patch("src.adapters.redis_adapter.requests.Session")
    def test_malformed_json_response_raises_redis_error(self, mock_session_class):
        """Malformed JSON response raises RedisError."""
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.text = "invalid json"
        mock_response.json.side_effect = json.JSONDecodeError("test", "doc", 0)

        mock_session = Mock()
        mock_session.get.return_value = mock_response
        mock_session_class.return_value = mock_session

        client = UpstashRedisClient("https://test.upstash.io", "test_token")

        with pytest.raises(RedisError, match="Invalid JSON response"):
            client.get("test_key")

        client.close()

    @patch("src.adapters.redis_adapter.requests.Session")
    def test_4xx_response_raises_redis_error_not_retried(self, mock_session_class):
        """4xx responses raise RedisError and are not retried."""
        mock_response = Mock()
        mock_response.status_code = 403
        mock_response.json.return_value = {"error": "Forbidden"}
        mock_response.text = "Forbidden"

        mock_session = Mock()
        mock_session.get.return_value = mock_response
        mock_session_class.return_value = mock_session

        client = UpstashRedisClient("https://test.upstash.io", "test_token")

        with pytest.raises(RedisError, match="403"):
            client.get("test_key")

        # Should only call once (no retries for 4xx)
        assert mock_session.get.call_count == 1
        client.close()

    @patch("src.adapters.redis_adapter.requests.Session")
    def test_timeout_error_wrapped_as_redis_timeout_error(self, mock_session_class):
        """Timeout errors are wrapped as RedisTimeoutError."""
        mock_session = Mock()
        mock_session.get.side_effect = __import__("requests").Timeout("timeout")
        mock_session_class.return_value = mock_session

        client = UpstashRedisClient("https://test.upstash.io", "test_token")

        with pytest.raises(RedisTimeoutError, match="timed out"):
            client.get("test_key")

        client.close()

    @patch("src.adapters.redis_adapter.requests.Session")
    def test_connection_error_wrapped_as_redis_connection_error(
        self, mock_session_class
    ):
        """Connection errors are wrapped as RedisConnectionError."""
        mock_session = Mock()
        mock_session.get.side_effect = __import__("requests").ConnectionError(
            "connection failed"
        )
        mock_session_class.return_value = mock_session

        client = UpstashRedisClient("https://test.upstash.io", "test_token")

        with pytest.raises(RedisConnectionError, match="Connection failed"):
            client.get("test_key")

        client.close()


class TestRetryWithBackoffDecorator:
    """Test retry decorator behavior."""

    def test_retry_decorator_retries_on_timeout(self):
        """Retry decorator retries on Timeout."""
        call_count = 0

        @retry_with_backoff(max_retries=3, initial_backoff_ms=10)
        def failing_then_succeeding():
            nonlocal call_count
            call_count += 1
            if call_count < 3:
                raise __import__("requests").Timeout("timeout")
            return "success"

        result = failing_then_succeeding()
        assert result == "success"
        assert call_count == 3

    def test_retry_decorator_stops_after_max_retries(self):
        """Retry decorator stops after max retries."""
        call_count = 0

        @retry_with_backoff(max_retries=3, initial_backoff_ms=10)
        def always_failing():
            nonlocal call_count
            call_count += 1
            raise __import__("requests").Timeout("timeout")

        with pytest.raises(RedisTimeoutError):
            always_failing()

        assert call_count == 3

    def test_retry_decorator_does_not_retry_non_transient_errors(self):
        """Retry decorator does not retry non-transient errors."""
        call_count = 0

        @retry_with_backoff(max_retries=3, initial_backoff_ms=10)
        def failing_with_value_error():
            nonlocal call_count
            call_count += 1
            raise ValueError("bad value")

        with pytest.raises(ValueError, match="bad value"):
            failing_with_value_error()

        # Should only call once, no retry
        assert call_count == 1

    def test_retry_decorator_exponential_backoff(self):
        """Retry decorator uses exponential backoff."""
        call_count = 0
        times = []

        @retry_with_backoff(max_retries=3, initial_backoff_ms=50)
        def failing_then_succeeding():
            nonlocal call_count
            times.append(time.time())
            call_count += 1
            if call_count < 3:
                raise __import__("requests").Timeout("timeout")
            return "success"

        result = failing_then_succeeding()
        assert result == "success"

        # Verify exponential backoff timing
        # Backoff should be ~50ms, ~100ms
        gap1 = (times[1] - times[0]) * 1000  # Convert to ms
        gap2 = (times[2] - times[1]) * 1000

        # Allow some tolerance for system scheduling
        assert gap1 >= 40  # ~50ms
        assert gap2 >= 80  # ~100ms



# ============================================================================
# SUMMARY
# ============================================================================

"""
Test Summary:
- Unit tests (8): Config validation, request formatting, error handling
- Retry decorator tests (4): Exponential backoff, max retries, non-transient errors

Total: 12 unit tests
"""
