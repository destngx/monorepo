import time
import logging
import threading
from enum import Enum
from typing import Optional

logger = logging.getLogger(__name__)

class CircuitBreakerState(Enum):
    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"

class CircuitBreaker:
    """Circuit breaker for Redis availability."""

    def __init__(self, failure_threshold: int = 3, backoff_ms: int = 100):
        self.state = CircuitBreakerState.CLOSED
        self.failure_count = 0
        self.failure_threshold = failure_threshold
        self.backoff_ms = backoff_ms
        self.last_failure_time: Optional[float] = None
        self._lock = threading.Lock()

    def record_success(self):
        with self._lock:
            if self.state == CircuitBreakerState.HALF_OPEN:
                logger.info("Circuit breaker: HALF_OPEN → CLOSED (success)")
                self.state = CircuitBreakerState.CLOSED
                self.failure_count = 0
            elif self.state == CircuitBreakerState.CLOSED:
                self.failure_count = 0

    def record_failure(self):
        with self._lock:
            self.failure_count += 1
            self.last_failure_time = time.monotonic()

            if self.state == CircuitBreakerState.HALF_OPEN:
                logger.warning(
                    f"Circuit breaker: HALF_OPEN → OPEN (failure {self.failure_count})"
                )
                self.state = CircuitBreakerState.OPEN
            elif (
                self.failure_count >= self.failure_threshold
                and self.state == CircuitBreakerState.CLOSED
            ):
                logger.warning(
                    f"Circuit breaker: CLOSED → OPEN ({self.failure_count} failures)"
                )
                self.state = CircuitBreakerState.OPEN

    def can_attempt(self) -> bool:
        with self._lock:
            if self.state == CircuitBreakerState.CLOSED:
                return True

            if self.state == CircuitBreakerState.OPEN:
                if self.last_failure_time is None:
                    return False

                elapsed_ms = (time.monotonic() - self.last_failure_time) * 1000
                if elapsed_ms >= self.backoff_ms:
                    logger.info("Circuit breaker: OPEN → HALF_OPEN (backoff expired)")
                    self.state = CircuitBreakerState.HALF_OPEN
                    return True
                return False

            if self.state == CircuitBreakerState.HALF_OPEN:
                return True

            return False

    def get_state(self) -> str:
        with self._lock:
            return self.state.value
