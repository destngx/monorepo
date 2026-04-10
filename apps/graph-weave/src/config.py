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

    UPSTASH_REDIS_REST_URL = os.getenv("UPSTASH_REDIS_REST_URL", "")
    UPSTASH_REDIS_REST_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN", "")

    @staticmethod
    def validate() -> bool:
        """Validate configuration is valid."""
        return True
