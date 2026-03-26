"""
Caching layer using Upstash Redis for vnstock-server.
Provides TTL-based caching for stock data API calls.
"""

import os
import json
import logging
from typing import Optional, Any, Dict
from datetime import datetime, timedelta
import httpx

logger = logging.getLogger(__name__)


class UpstashRedisClient:
    """
    HTTP client for Upstash Redis REST API.
    Upstash provides serverless Redis without managing infrastructure.
    """

    def __init__(self, url: str, token: str):
        self.url = url.rstrip("/")
        self.token = token
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        }

    async def get(self, key: str) -> Optional[str]:
        """Get value from Redis."""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    f"{self.url}/get/{key}",
                    headers=self.headers,
                    timeout=5.0,
                )
                response.raise_for_status()
                result = response.json()
                if result.get("result"):
                    return result["result"]
                return None
        except Exception as e:
            logger.warning(f"Redis GET error for {key}: {e}")
            return None

    async def set(self, key: str, value: str, ex: Optional[int] = None) -> bool:
        """Set value in Redis with optional TTL (in seconds)."""
        try:
            async with httpx.AsyncClient() as client:
                cmd = ["SET", key, value]
                if ex:
                    cmd.extend(["EX", str(ex)])

                response = await client.post(
                    f"{self.url}/exec",
                    headers=self.headers,
                    json={"commands": [cmd]},
                    timeout=5.0,
                )
                response.raise_for_status()
                return True
        except Exception as e:
            logger.warning(f"Redis SET error for {key}: {e}")
            return False

    async def delete(self, key: str) -> bool:
        """Delete key from Redis."""
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.url}/exec",
                    headers=self.headers,
                    json={"commands": [["DEL", key]]},
                    timeout=5.0,
                )
                response.raise_for_status()
                return True
        except Exception as e:
            logger.warning(f"Redis DEL error for {key}: {e}")
            return False


class CacheManager:
    """
    Manages caching for vnstock API responses.
    Uses Upstash Redis for distributed, serverless caching.
    """

    DEFAULT_TTL = {
        "quote": 60,
        "listing": 3600,
        "historical": 3600,
    }

    def __init__(
        self, redis_url: Optional[str] = None, redis_token: Optional[str] = None
    ):
        self.redis_url = redis_url or os.getenv("UPSTASH_REDIS_REST_URL")
        self.redis_token = redis_token or os.getenv("UPSTASH_REDIS_REST_TOKEN")
        self.enabled = bool(self.redis_url and self.redis_token)

        if self.enabled:
            self.client = UpstashRedisClient(self.redis_url, self.redis_token)
            logger.info("✓ Cache layer initialized (Upstash Redis)")
        else:
            self.client = None
            logger.warning("⚠ Cache disabled (Upstash Redis not configured)")

    @staticmethod
    def _make_key(endpoint: str, params: Dict[str, Any]) -> str:
        """Generate consistent cache key from endpoint and parameters."""
        param_str = ":".join(f"{k}={v}" for k, v in sorted(params.items()))
        return f"vnstock:{endpoint}:{param_str}"

    async def get(self, endpoint: str, params: Dict[str, Any]) -> Optional[Any]:
        """Get cached data if available."""
        if not self.enabled:
            return None

        key = self._make_key(endpoint, params)
        try:
            cached = await self.client.get(key)
            if cached:
                logger.debug(f"✓ Cache hit: {key}")
                return json.loads(cached)
            logger.debug(f"✗ Cache miss: {key}")
            return None
        except Exception as e:
            logger.warning(f"Cache retrieval error: {e}")
            return None

    async def set(
        self,
        endpoint: str,
        params: Dict[str, Any],
        data: Any,
        ttl: Optional[int] = None,
    ) -> bool:
        """Set cached data with TTL."""
        if not self.enabled:
            return False

        key = self._make_key(endpoint, params)
        ttl = ttl or self.DEFAULT_TTL.get(endpoint, 300)

        try:
            await self.client.set(key, json.dumps(data), ex=ttl)
            logger.debug(f"✓ Cached: {key} (TTL: {ttl}s)")
            return True
        except Exception as e:
            logger.warning(f"Cache storage error: {e}")
            return False

    async def invalidate(
        self, endpoint: str, params: Optional[Dict[str, Any]] = None
    ) -> bool:
        """Invalidate cache for a specific endpoint or all matching keys."""
        if not self.enabled:
            return False

        if params:
            key = self._make_key(endpoint, params)
            return await self.client.delete(key)
        return True


class CacheConfig:
    """Configuration for cache behavior."""

    QUOTE_TTL = 60
    LISTING_TTL = 3600
    HISTORICAL_TTL = 3600
    DEFAULT_TTL = 300

    CACHE_ENABLED = os.getenv("CACHE_ENABLED", "true").lower() == "true"
    REDIS_URL = os.getenv("UPSTASH_REDIS_REST_URL")
    REDIS_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN")

    @classmethod
    def validate(cls) -> bool:
        """Validate cache configuration."""
        if not cls.CACHE_ENABLED:
            return True

        if not cls.REDIS_URL or not cls.REDIS_TOKEN:
            logger.warning("Cache enabled but Redis credentials missing")
            return False

        return True
