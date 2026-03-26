"""
Rate limiting and quota management for vnstock API requests.
Prevents hitting Community tier limits (60 req/min, 10k req/day).
"""

import os
import json
import logging
import asyncio
from datetime import datetime, timedelta
from typing import Optional, Any, Dict
from dataclasses import dataclass, asdict
from enum import Enum

logger = logging.getLogger(__name__)


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
    Manages API request rate limiting for Community tier (60 req/min, 10k req/day).

    Features:
    - Request queue with exponential backoff
    - Per-minute and per-day quota tracking
    - Automatic backoff on 429 responses
    - Circuit breaker on quota exhaustion
    - Persistent quota tracking via environment
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

        # Request tracking
        self.requests_minute: list[
            float
        ] = []  # Timestamps of requests in current minute
        self.requests_day: list[float] = []  # Timestamps of requests in current day

        # Backoff state
        self.backoff_until: Optional[float] = None
        self.backoff_multiplier = 1.0
        self.max_backoff_seconds = 300  # 5 minutes max

        # Request queue for fair distribution
        self.request_queue: asyncio.Queue = asyncio.Queue()
        self.queue_processor_task: Optional[asyncio.Task] = None

        # Load persisted quota from environment (if available)
        self._load_persisted_quota()

        logger.info(
            f"✓ Rate limiter initialized: {self.tier} ({self.per_minute} req/min, {self.per_day} req/day)"
        )

    def _load_persisted_quota(self):
        """Load quota tracking from environment if available."""
        try:
            quota_data = os.getenv("RATE_LIMIT_QUOTA")
            if quota_data:
                data = json.loads(quota_data)
                # Parse and restore timestamps
                now = datetime.now()
                minute_ago = (now - timedelta(minutes=1)).timestamp()
                day_ago = (now - timedelta(days=1)).timestamp()

                # Restore requests from current minute
                self.requests_minute = [
                    ts for ts in data.get("minute_requests", []) if ts > minute_ago
                ]
                # Restore requests from current day
                self.requests_day = [
                    ts for ts in data.get("day_requests", []) if ts > day_ago
                ]
                logger.debug(
                    f"Restored quota: {len(self.requests_minute)} min, {len(self.requests_day)} day"
                )
        except Exception as e:
            logger.warning(f"Could not load persisted quota: {e}")

    def _persist_quota(self):
        """Persist quota tracking to environment."""
        try:
            data = {
                "minute_requests": self.requests_minute,
                "day_requests": self.requests_day,
                "timestamp": datetime.now().isoformat(),
            }
            os.environ["RATE_LIMIT_QUOTA"] = json.dumps(data)
        except Exception as e:
            logger.warning(f"Could not persist quota: {e}")

    def _cleanup_old_requests(self):
        """Remove requests outside tracking window."""
        now = datetime.now().timestamp()
        minute_ago = now - 60
        day_ago = now - 86400

        self.requests_minute = [ts for ts in self.requests_minute if ts > minute_ago]
        self.requests_day = [ts for ts in self.requests_day if ts > day_ago]

    async def check_available(self) -> bool:
        """Check if a request can be made immediately."""
        self._cleanup_old_requests()

        # Check if in backoff
        if self.backoff_until:
            now = datetime.now().timestamp()
            if now < self.backoff_until:
                return False

        # Check minute limit
        if len(self.requests_minute) >= self.per_minute:
            logger.warning(f"⚠ Per-minute limit reached ({self.per_minute})")
            return False

        # Check day limit
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
        self._cleanup_old_requests()

        # Circuit breaker: if day limit exhausted, fail fast
        if len(self.requests_day) >= self.per_day:
            logger.error(
                "🔴 CIRCUIT BREAKER: Daily quota exhausted. No more requests possible."
            )
            return False

        # Wait out backoff period with exponential backoff
        while self.backoff_until:
            now = datetime.now().timestamp()
            if now >= self.backoff_until:
                self.backoff_until = None
                self.backoff_multiplier = 1.0
                break

            wait_seconds = min(self.backoff_until - now, self.max_backoff_seconds)
            logger.warning(f"⏳ Backoff in effect, waiting {wait_seconds:.1f}s")
            await asyncio.sleep(wait_seconds)

        # Wait for per-minute slot
        while len(self.requests_minute) >= self.per_minute:
            # Find when the oldest request leaves the window
            oldest = min(self.requests_minute)
            wait_seconds = 60 - (datetime.now().timestamp() - oldest)
            if wait_seconds > 0:
                logger.info(
                    f"⏳ Rate limit queue: waiting {wait_seconds:.1f}s for per-minute slot"
                )
                await asyncio.sleep(min(wait_seconds, 1))  # Sleep in 1s chunks
            self._cleanup_old_requests()

        # Acquire slot
        now = datetime.now().timestamp()
        self.requests_minute.append(now)
        self.requests_day.append(now)
        self._persist_quota()

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
        # Handle 429 (rate limited) responses
        if status_code == 429:
            self._handle_rate_limited()
        # Handle 503 (service unavailable)
        elif status_code == 503:
            self._handle_backoff(seconds=5)

    def _handle_rate_limited(self):
        """Handle 429 response from API."""
        logger.warning("⚠ Received 429 (rate limited) from vnstock API")
        # Apply exponential backoff
        backoff_seconds = min(
            5 * (2**self.backoff_multiplier), self.max_backoff_seconds
        )
        self._handle_backoff(seconds=backoff_seconds)

    def _handle_backoff(self, seconds: float):
        """Apply backoff delay."""
        self.backoff_until = datetime.now().timestamp() + seconds
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

        # Calculate reset times
        if self.requests_minute:
            oldest_minute = datetime.fromtimestamp(min(self.requests_minute))
            minute_reset = oldest_minute + timedelta(minutes=1)
        else:
            minute_reset = now + timedelta(minutes=1)

        if self.requests_day:
            oldest_day = datetime.fromtimestamp(min(self.requests_day))
            day_reset = oldest_day + timedelta(days=1)
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
        status = self.get_status()
        return asdict(status)
