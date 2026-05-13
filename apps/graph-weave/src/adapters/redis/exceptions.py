class RedisError(Exception):
    """Base exception for Redis operations."""
    pass

class RedisTimeoutError(RedisError):
    """Raised when Redis operation times out."""
    pass

class RedisConnectionError(RedisError):
    """Raised when connection to Redis fails."""
    pass
