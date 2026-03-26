"""
Configuration management for vnstock-server.
Handles environment variables and validation.
"""

import os
from typing import Optional
from functools import lru_cache


class VnstockConfig:
    """Configuration for vnstock API and server settings."""

    @staticmethod
    @lru_cache(maxsize=1)
    def get_api_key() -> str:
        """
        Get VNSTOCK_API_KEY from environment.

        Returns:
            str: API key for authenticated vnstock requests

        Raises:
            ValueError: If VNSTOCK_API_KEY is not set
        """
        api_key = os.getenv("VNSTOCK_API_KEY")
        if not api_key or not api_key.strip():
            raise ValueError(
                "VNSTOCK_API_KEY environment variable is required but not set. "
                "Set it in .env.local or your environment."
            )
        return api_key.strip()

    @staticmethod
    def validate() -> bool:
        """
        Validate that API key is properly configured.

        Returns:
            bool: True if valid, raises exception otherwise

        Raises:
            ValueError: If API key is invalid
        """
        try:
            VnstockConfig.get_api_key()
            return True
        except ValueError as e:
            raise ValueError(f"Configuration validation failed: {str(e)}")


class ServerConfig:
    """Server configuration."""

    DEBUG = os.getenv("DEBUG", "false").lower() == "true"
    HOST = os.getenv("HOST", "0.0.0.0")
    PORT = int(os.getenv("PORT", 8000))

    # Default data source for vnstock (VCI or KBS)
    DEFAULT_SOURCE = os.getenv("VNSTOCK_SOURCE", "VCI")
