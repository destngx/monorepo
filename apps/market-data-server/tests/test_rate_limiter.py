import asyncio
from unittest.mock import patch

from src.rate_limiter import RateLimiter


def test_acquire_waits_for_the_oldest_request_to_leave_the_window():
    limiter = RateLimiter("COMMUNITY")
    limiter.per_minute = 2
    clock = [1000.0]
    sleeps = []

    async def fake_sleep(seconds):
        sleeps.append(seconds)
        clock[0] += seconds

    async def run():
        with patch("src.rate_limiter.time.time", side_effect=lambda: clock[0]), patch(
            "src.rate_limiter.asyncio.sleep", side_effect=fake_sleep
        ):
            await limiter.acquire()
            clock[0] += 10
            await limiter.acquire()
            await limiter.acquire()

    asyncio.run(run())
    assert sleeps == [50.0]
    assert len(limiter.requests_minute) == 2
    assert len(limiter.requests_day) == 3


def test_acquire_fails_fast_when_the_daily_quota_is_exhausted():
    limiter = RateLimiter("COMMUNITY")
    limiter.per_day = 1
    assert asyncio.run(limiter.acquire()) is True
    assert asyncio.run(limiter.acquire()) is False
