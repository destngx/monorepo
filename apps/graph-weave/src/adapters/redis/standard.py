import json
from typing import Optional, Dict, Any, List


try:
    import redis
except ImportError:
    redis = None

class RedisAdapter:
    """Standard TCP-based Redis client using redis-py."""
    def __init__(self, url: str, token: str):
        self.url = url
        self.token = token
        if redis:
            self.client = redis.from_url(url, password=token, decode_responses=True)
        else:
            self.client = None

    @classmethod
    def from_env(cls, url: Optional[str] = None, token: Optional[str] = None):
        if not url or not token:
            raise ValueError("Redis URL and token must be provided")
        if url.startswith("https://"):
            url = "rediss://" + url[len("https://") :]
        elif url.startswith("http://"):
            url = "redis://" + url[len("http://") :]
        return cls(url, token)

    def set(self, key: str, value: Any, ex: Optional[int] = None, **kwargs) -> None:
        if not self.client:
            return
        self.client.set(key, json.dumps(value), ex=ex, **kwargs)

    def get(self, key: str) -> Optional[Any]:
        if not self.client:
            return None
        value = self.client.get(key)
        return json.loads(value) if value else None

    def delete(self, key: str) -> bool:
        if not self.client:
            return False
        return bool(self.client.delete(key))

    def exists(self, key: str) -> bool:
        if not self.client:
            return False
        return bool(self.client.exists(key))

    def rpush(self, key: str, value: Any) -> int:
        if not self.client:
            return 0
        return self.client.rpush(key, json.dumps(value))

    def lpush(self, key: str, value: Any) -> int:
        if not self.client:
            return 0
        return self.client.lpush(key, json.dumps(value))

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        if not self.client:
            return []
        items = self.client.lrange(key, start, end)
        return [json.loads(item) for item in items]

    def ltrim(self, key: str, start: int, end: int) -> bool:
        if not self.client:
            return False
        return bool(self.client.ltrim(key, start, end))

    def ttl(self, key: str) -> int:
        if not self.client:
            return -2
        return self.client.ttl(key)

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        if not self.client:
            return {}
        values = self.client.mget(keys)
        return {
            keys[i]: json.loads(values[i])
            for i in range(len(keys))
            if values[i] is not None
        }

    def hset(self, key: str, field: str, value: Any) -> int:
        if not self.client:
            return 0
        return self.client.hset(key, field, json.dumps(value))

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        if not self.client:
            return 0
        return self.client.hsetnx(key, field, json.dumps(value))

    def hget(self, key: str, field: str) -> Optional[Any]:
        if not self.client:
            return None
        value = self.client.hget(key, field)
        return json.loads(value) if value else None

    def hdel(self, key: str, field: str) -> int:
        if not self.client:
            return 0
        return self.client.hdel(key, field)

    def hgetall(self, key: str) -> Dict[str, Any]:
        if not self.client:
            return {}
        data = self.client.hgetall(key)
        return {k: json.loads(v) for k, v in data.items()}

    def sadd(self, key: str, *values: Any) -> int:
        if not self.client:
            return 0
        return self.client.sadd(key, *[json.dumps(value) for value in values])

    def smembers(self, key: str) -> set:
        if not self.client:
            return set()
        return {json.loads(value) for value in self.client.smembers(key)}

    def srem(self, key: str, *values: Any) -> int:
        if not self.client:
            return 0
        return self.client.srem(key, *[json.dumps(value) for value in values])

    def sinter(self, *keys: str) -> set:
        if not self.client or not keys:
            return set()
        return {json.loads(value) for value in self.client.sinter(*keys)}

    def keys(self, pattern: str) -> List[str]:
        if not self.client:
            return []
        return self.client.keys(pattern)

    def clear(self):
        if self.client:
            self.client.flushdb()

    def close(self):
        if self.client:
            self.client.close()
