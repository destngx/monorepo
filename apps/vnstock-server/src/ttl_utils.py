"""
TTL calculation utility for cache expiration.
Calculates seconds until next 5am GMT+7 to ensure fresh data daily.
"""

from datetime import datetime, timedelta, timezone
import logging

logger = logging.getLogger(__name__)

# GMT+7 timezone offset
GMT_PLUS_7 = timezone(timedelta(hours=7))

# Default max TTL: 1 day (86400 seconds)
DEFAULT_MAX_TTL = 86400


def calculate_ttl_until_next_5am(max_ttl: int = DEFAULT_MAX_TTL) -> int:
    """
    Calculate seconds until the next 5:00 AM GMT+7.

    Rules:
    - If current time is before 5am GMT+7: TTL = seconds until 5am today
    - If current time is after 5am GMT+7: TTL = seconds until 5am tomorrow
    - Never exceeds max_ttl (default 1 day)

    Args:
        max_ttl: Maximum TTL in seconds (default: 86400 = 1 day)

    Returns:
        TTL in seconds, capped at max_ttl

    Examples:
        >>> # If it's 03:00 GMT+7, returns ~7200 (2 hours until 5am)
        >>> # If it's 06:00 GMT+7, returns ~82800 (23 hours until next 5am)
    """
    now = datetime.now(GMT_PLUS_7)

    # Today at 5:00 AM GMT+7
    reset_time = now.replace(hour=5, minute=0, second=0, microsecond=0)

    # If we've already passed 5am today, next reset is tomorrow
    if now >= reset_time:
        reset_time += timedelta(days=1)

    ttl = int((reset_time - now).total_seconds())

    # Ensure TTL doesn't exceed max_ttl
    ttl = min(ttl, max_ttl)

    logger.debug(
        f"[TTL] Current time: {now.isoformat()} | "
        f"Next reset: {reset_time.isoformat()} | "
        f"TTL: {ttl}s ({ttl / 3600:.1f}h)"
    )

    return ttl


def get_cache_ttl(data_type: str = "default", max_ttl: int = DEFAULT_MAX_TTL) -> int:
    """
    Get TTL for a specific data type.
    All types use dynamic calculation until next 5am GMT+7, capped at 1 day.

    Args:
        data_type: Type of data (quote, listing, historical, etc.)
        max_ttl: Maximum TTL in seconds

    Returns:
        TTL in seconds
    """
    return calculate_ttl_until_next_5am(max_ttl=max_ttl)
