"""
Configuration management for graph-weave.
Handles environment variables and validation.
"""

import os
from typing import Optional
from functools import lru_cache

# Lightweight dynamic dotenv loader
def _load_env_file(filepath: str) -> None:
    if not os.path.exists(filepath):
        return
    try:
        with open(filepath, "r") as f:
            for line in f:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                if "=" in line:
                    key, val = line.split("=", 1)
                    key = key.strip()
                    val = val.strip().strip("'\"")
                    if key and key not in os.environ:
                        os.environ[key] = val
    except Exception:
        pass

# Automatically load .env and .env.local from both app and monorepo roots
for env_name in [".env", ".env.local", "../.env", "../.env.local", "../../.env", "../../.env.local", "../../../.env", "../../../.env.local"]:
    _abs_path = os.path.abspath(os.path.join(os.path.dirname(__file__), env_name))
    _load_env_file(_abs_path)


class GraphWeaveConfig:
    """Configuration for GraphWeave application."""

    DEBUG = os.getenv("DEBUG", "false").lower() == "true"
    HOST = os.getenv("HOST", "0.0.0.0")
    PORT = int(os.getenv("PORT", 8001))
    SCHEDULER_ENABLED = os.getenv("SCHEDULER_ENABLED", "true").lower() == "true"

    UPSTASH_REDIS_REST_URL = os.getenv("UPSTASH_REDIS_REST_URL", "")
    UPSTASH_REDIS_REST_TOKEN = os.getenv("UPSTASH_REDIS_REST_TOKEN", "")

    DEFAULT_PROVIDER = os.getenv("DEFAULT_PROVIDER", "github-copilot")
    DEFAULT_MODEL = os.getenv("DEFAULT_MODEL", "gpt-5.4-mini")
    DEFAULT_LARGE_CONTEXT_MODEL = os.getenv("DEFAULT_LARGE_CONTEXT_MODEL", "gpt-5.4")
    DEFAULT_REASONING_EFFORT = os.getenv("DEFAULT_REASONING_EFFORT", "low")

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
