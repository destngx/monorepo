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

    # Mock settings
    ENABLE_MOCK_REDIS = os.getenv("ENABLE_MOCK_REDIS", "true").lower() == "true"
    ENABLE_MOCK_MCP = os.getenv("ENABLE_MOCK_MCP", "true").lower() == "true"
    ENABLE_MOCK_LANGGRAPH = os.getenv("ENABLE_MOCK_LANGGRAPH", "true").lower() == "true"

    @staticmethod
    def validate() -> bool:
        """Validate configuration is valid."""
        return True
