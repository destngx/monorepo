"""
Configuration management for graph-weave.
Handles environment variables and validation.
"""

import os
from typing import Optional
from functools import lru_cache


class GraphWeaveConfig:
    """Configuration for GraphWeave application."""

    DEBUG = os.getenv("DEBUG", "false").lower() == "true"
    HOST = os.getenv("HOST", "0.0.0.0")
    PORT = int(os.getenv("PORT", 8001))
    SCHEDULER_ENABLED = os.getenv("SCHEDULER_ENABLED", "true").lower() == "true"

    UPSTASH_REDIS_REST_URL = os.getenv("UPSTASH_REDIS_REST_URL", "")
    UPSTASH_REDIS_REST_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN", "")

    @staticmethod
    def validate() -> bool:
        """Validate required configuration is present."""
        # Redis is optional for local development/testing
        # In production, UPSTASH_REDIS_REST_URL and UPSTASH_REDIS_REST_TOKEN should be set
        if (
            GraphWeaveConfig.UPSTASH_REDIS_REST_URL
            and not GraphWeaveConfig.UPSTASH_REDIS_REST_TOKEN
        ):
            raise ValueError(
                "UPSTASH_REDIS_REST_TOKEN is required when UPSTASH_REDIS_REST_URL is set"
            )
        return True
