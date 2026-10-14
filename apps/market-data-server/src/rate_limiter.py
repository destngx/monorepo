"""
Rate limiting and quota management for vnstock API requests.
Prevents hitting Community tier limits (60 req/min, 10k req/day).
"""

import time
import logging
import asyncio
from collections import deque
from datetime import datetime, timedelta
from typing import Optional, Any, Dict
from dataclasses import dataclass, asdict
from enum import Enum

logger = logging.getLogger(__name__)

MINUTE = 60
DAY = 86400


class RateLimitTier(Enum):
    """vnstock API rate limit tiers."""

    COMMUNITY = {"per_minute": 60, "per_day": 10000, "display": "Community"}
    BRONZE = {"per_minute": 180, "per_day": 50000, "display": "Bronze"}
    SILVER = {"per_minute": 300, "per_day": 100000, "display": "Silver"}
    GOLDEN = {"per_minute": 500, "per_day": 150000, "display": "Golden"}
    DIAMOND = {"per_minute": 600, "per_day": 180000, "display": "Diamond"}


@dataclass
class RateLimitStatus:
    """Current rate limit status."""

    tier: str
    requests_per_minute: int
    requests_per_day: int
    current_minute_requests: int
    current_day_requests: int
    remaining_minute: int
    remaining_day: int
    minute_reset_at: str
    day_reset_at: str
    is_rate_limited: bool
    backoff_until: Optional[str] = None


class RateLimiter:
    """
    Sliding-window limiter for upstream vnstock calls.

    Features:
    - Per-minute and per-day quota tracking (O(1) amortised via deques)
    - Waiters sleep exactly until the next slot frees up
    - Exponential backoff on 429 responses
    - Circuit breaker on daily quota exhaustion
    """

    def __init__(self, tier: str = "COMMUNITY"):
        """
        Initialize rate limiter.

        Args:
            tier: Rate limit tier (COMMUNITY, BRONZE, SILVER, GOLDEN, DIAMOND)
        """
        try:
            tier_enum = RateLimitTier[tier.upper()]
            tier_config = tier_enum.value
            self.tier = tier_enum.name
            self.per_minute = tier_config["per_minute"]
            self.per_day = tier_config["per_day"]
        except KeyError:
            logger.warning(f"Unknown tier {tier}, defaulting to COMMUNITY")
            self.tier = "COMMUNITY"
            self.per_minute = 60
            self.per_day = 10000

        # Wall-clock timestamps of granted requests, oldest first.
        self.requests_minute: deque[float] = deque()
        self.requests_day: deque[float] = deque()

        # Backoff state
        self.backoff_until: Optional[float] = None
        self.backoff_multiplier = 1.0
        self.max_backoff_seconds = 300  # 5 minutes max

        logger.info(
            f"✓ Rate limiter initialized: {self.tier} ({self.per_minute} req/min, {self.per_day} req/day)"
        )

    def _cleanup_old_requests(self, now: Optional[float] = None):
        """Remove requests outside tracking window."""
        now = time.time() if now is None else now
        while self.requests_minute and self.requests_minute[0] <= now - MINUTE:
            self.requests_minute.popleft()
        while self.requests_day and self.requests_day[0] <= now - DAY:
            self.requests_day.popleft()

    async def check_available(self) -> bool:
        """Check if a request can be made immediately."""
        now = time.time()
        self._cleanup_old_requests(now)

        if self.backoff_until and now < self.backoff_until:
            return False

        if len(self.requests_minute) >= self.per_minute:
            logger.warning(f"⚠ Per-minute limit reached ({self.per_minute})")
            return False

        if len(self.requests_day) >= self.per_day:
            logger.error(f"🔴 Per-day limit EXHAUSTED ({self.per_day})")
            return False

        return True

    async def acquire(self) -> bool:
        """
        Wait for rate limit availability and acquire a request slot.

        Returns:
            True if request can proceed, False if rate limited (circuit breaker)
        """
        while True:
            now = time.time()
            self._cleanup_old_requests(now)

            # Circuit breaker: if day limit exhausted, fail fast
            if len(self.requests_day) >= self.per_day:
                logger.error(
                    "🔴 CIRCUIT BREAKER: Daily quota exhausted. No more requests possible."
                )
                return False

            if self.backoff_until:
                if now < self.backoff_until:
                    wait_seconds = min(self.backoff_until - now, self.max_backoff_seconds)
                    logger.warning(f"⏳ Backoff in effect, waiting {wait_seconds:.1f}s")
                    await asyncio.sleep(wait_seconds)
                    continue
                self.backoff_until = None
                self.backoff_multiplier = 1.0

            if len(self.requests_minute) >= self.per_minute:
                wait_seconds = self.requests_minute[0] + MINUTE - now
                logger.info(
                    f"⏳ Rate limit queue: waiting {wait_seconds:.1f}s for per-minute slot"
                )
                await asyncio.sleep(max(wait_seconds, 0.01))
                continue

            # No await between the checks above and this append, so concurrent
            # coroutines cannot overshoot the quota.
            self.requests_minute.append(now)
            self.requests_day.append(now)
            return True

    async def record_response(
        self, status_code: int, response_headers: Optional[Dict[str, str]] = None
    ):
        """
        Record API response to update rate limit state.

        Args:
            status_code: HTTP status code from vnstock API
            response_headers: Response headers (may contain X-RateLimit-*)
        """
        if status_code == 429:
            self._handle_rate_limited()
        elif status_code == 503:
            self._handle_backoff(seconds=5)

    def _handle_rate_limited(self):
        """Handle 429 response from API."""
        logger.warning("⚠ Received 429 (rate limited) from vnstock API")
        backoff_seconds = min(
            5 * (2**self.backoff_multiplier), self.max_backoff_seconds
        )
        self._handle_backoff(seconds=backoff_seconds)

    def _handle_backoff(self, seconds: float):
        """Apply backoff delay."""
        self.backoff_until = time.time() + seconds
        self.backoff_multiplier += 0.5
        logger.warning(
            f"📍 Backoff applied: {seconds}s (until {datetime.fromtimestamp(self.backoff_until)})"
        )

    def get_status(self) -> RateLimitStatus:
        """Get current rate limit status."""
        self._cleanup_old_requests()
        now = datetime.now()

        remaining_minute = max(0, self.per_minute - len(self.requests_minute))
        remaining_day = max(0, self.per_day - len(self.requests_day))

        if self.requests_minute:
            minute_reset = datetime.fromtimestamp(self.requests_minute[0]) + timedelta(minutes=1)
        else:
            minute_reset = now + timedelta(minutes=1)

        if self.requests_day:
            day_reset = datetime.fromtimestamp(self.requests_day[0]) + timedelta(days=1)
        else:
            day_reset = now + timedelta(days=1)

        return RateLimitStatus(
            tier=self.tier,
            requests_per_minute=self.per_minute,
            requests_per_day=self.per_day,
            current_minute_requests=len(self.requests_minute),
            current_day_requests=len(self.requests_day),
            remaining_minute=remaining_minute,
            remaining_day=remaining_day,
            minute_reset_at=minute_reset.isoformat(),
            day_reset_at=day_reset.isoformat(),
            is_rate_limited=self.backoff_until is not None,
            backoff_until=datetime.fromtimestamp(self.backoff_until).isoformat()
            if self.backoff_until is not None
            else None,
        )

    def get_status_dict(self) -> Dict[str, Any]:
        """Get status as dictionary (for JSON response)."""
        return asdict(self.get_status())
