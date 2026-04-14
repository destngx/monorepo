from typing import Optional, Dict, Any, List
import json

try:
    import redis
except ImportError:  # pragma: no cover - optional runtime dependency
    redis = None


class MockRedisAdapter:
    def __init__(self):
        self._store: Dict[str, Any] = {}

    def set(self, key: str, value: Any, ex: Optional[int] = None, **kwargs) -> None:
        self._store[key] = value

    def get(self, key: str) -> Optional[Any]:
        return self._store.get(key)

    def delete(self, key: str) -> None:
        if key in self._store:
            del self._store[key]

    def exists(self, key: str) -> bool:
        return key in self._store

    def clear(self) -> None:
        self._store.clear()

    def rpush(self, key: str, value: Any) -> int:
        if key not in self._store:
            self._store[key] = []
        self._store[key].append(value)
        return len(self._store[key])

    def lpush(self, key: str, value: Any) -> int:
        if key not in self._store:
            self._store[key] = []
        self._store[key].insert(0, value)
        return len(self._store[key])

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        items = self._store.get(key, [])
        if not isinstance(items, list):
            return []
        if end == -1:
            return items[start:]
        return items[start : end + 1]

    def ltrim(self, key: str, start: int, end: int) -> bool:
        if key in self._store and isinstance(self._store[key], list):
            if end == -1:
                self._store[key] = self._store[key][start:]
            else:
                self._store[key] = self._store[key][start : end + 1]
            return True
        return False

    def ttl(self, key: str) -> int:
        return -1 if key in self._store else -2

    def close(self) -> None:
        pass

    def _build_versioned_key(
        self, namespace: str, tenant_id: str, skill_id: str, version: str
    ) -> str:
        return f"{namespace}:{tenant_id}:{skill_id}:{version}"

    def set_versioned(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
        value: Any,
    ) -> None:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        self.set(key, value)

    def get_versioned(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
    ) -> Optional[Any]:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        return self.get(key)

    def get_versioned_with_fallback(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: Optional[str],
    ) -> Optional[Any]:
        if version is not None:
            return self.get_versioned(namespace, tenant_id, skill_id, version)
        return self.get_versioned(namespace, tenant_id, skill_id, "latest")

    def get_versioned_with_rebuild(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
        source_loader: Any,
    ) -> Any:
        cached = self.get_versioned(namespace, tenant_id, skill_id, version)
        if cached is not None:
            return cached

        entry = source_loader.load_from_source(tenant_id, skill_id, version)
        self.set_versioned(namespace, tenant_id, skill_id, version, entry)
        return entry


class RedisAdapter(MockRedisAdapter):
    def __init__(self, client: Any):
        super().__init__()
        self._client = client

    @classmethod
    def from_env(cls, url: str, token: str) -> "RedisAdapter":
        if redis is None:
            raise RuntimeError("redis package is not installed")

        # Handle Upstash REST URLs (convert https:// to rediss://)
        if url.startswith("https://"):
            url = "rediss://" + url[8:]
        elif url.startswith("http://"):
            url = "redis://" + url[7:]

        client = redis.Redis.from_url(url, password=token, decode_responses=True)
        return cls(client)

    def set(self, key: str, value: Any, ex: Optional[int] = None, **kwargs) -> None:
        super().set(key, value, ex=ex, **kwargs)
        if hasattr(self._client, "set"):
            # Ensure booleans are serialized as strings ('true'/'false') because real redis client fails on bool
            serialized = json.dumps(value) if not isinstance(value, (str, int, float)) or isinstance(value, bool) else value
            self._client.set(key, serialized, ex=ex)

    def get(self, key: str) -> Optional[Any]:
        if hasattr(self._client, "get"):
            value = self._client.get(key)
            if value is not None:
                try:
                    return json.loads(value)
                except (json.JSONDecodeError, TypeError):
                    return value
        return super().get(key)

    def delete(self, key: str) -> None:
        super().delete(key)
        if hasattr(self._client, "delete"):
            self._client.delete(key)

    def exists(self, key: str) -> bool:
        if hasattr(self._client, "exists"):
            return bool(self._client.exists(key))
        return super().exists(key)

    def rpush(self, key: str, value: Any) -> int:
        super().rpush(key, value)
        if hasattr(self._client, "rpush"):
            serialized = json.dumps(value) if not isinstance(value, (str, int, float, bool)) else value
            return self._client.rpush(key, serialized)
        return len(self._store.get(key, []))

    def lpush(self, key: str, value: Any) -> int:
        super().lpush(key, value)
        if hasattr(self._client, "lpush"):
            serialized = json.dumps(value) if not isinstance(value, (str, int, float, bool)) else value
            return self._client.lpush(key, serialized)
        return len(self._store.get(key, []))

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        if hasattr(self._client, "lrange"):
            items = self._client.lrange(key, start, end)
            return [json.loads(i) if isinstance(i, str) and (i.startswith("{") or i.startswith("[")) else i for i in items]
        return super().lrange(key, start, end)

    def ltrim(self, key: str, start: int, end: int) -> bool:
        super().ltrim(key, start, end)
        if hasattr(self._client, "ltrim"):
            return bool(self._client.ltrim(key, start, end))
        return True

    def ttl(self, key: str) -> int:
        if hasattr(self._client, "ttl"):
            return self._client.ttl(key)
        return super().ttl(key)

    def close(self) -> None:
        if hasattr(self._client, "close"):
            self._client.close()
