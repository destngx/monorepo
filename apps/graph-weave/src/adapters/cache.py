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

    def delete(self, key: str) -> bool:
        if key in self._store:
            del self._store[key]
            return True
        return False

    def exists(self, key: str) -> bool:
        return key in self._store

    def clear(self) -> None:
        self._store.clear()

    def rpush(self, key: str, value: Any) -> int:
        if key not in self._store:
            self._store[key] = []
        if not isinstance(self._store[key], list):
            self._store[key] = [self._store[key]]
        self._store[key].append(value)
        return len(self._store[key])

    def lpush(self, key: str, value: Any) -> int:
        if key not in self._store:
            self._store[key] = []
        if not isinstance(self._store[key], list):
            self._store[key] = [self._store[key]]
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

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        return {k: self._store[k] for k in keys if k in self._store}

    def hset(self, key: str, field: str, value: Any) -> int:
        if key not in self._store or not isinstance(self._store[key], dict):
            self._store[key] = {}
        self._store[key][field] = value
        return 1

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        if key not in self._store or not isinstance(self._store[key], dict):
            self._store[key] = {}
        if field in self._store[key]:
            return 0
        self._store[key][field] = value
        return 1

    def hget(self, key: str, field: str) -> Optional[Any]:
        if key in self._store and isinstance(self._store[key], dict):
            return self._store[key].get(field)
        return None

    def hdel(self, key: str, field: str) -> int:
        if key in self._store and isinstance(self._store[key], dict):
            if field in self._store[key]:
                del self._store[key][field]
                return 1
        return 0

    def hgetall(self, key: str) -> Dict[str, Any]:
        if key in self._store and isinstance(self._store[key], dict):
            return dict(self._store[key])
        return {}

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
        self, namespace: str, tenant_id: str, skill_id: str, version: str
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
        resolved_version = version or "latest"
        return self.get_versioned(
            namespace, tenant_id, skill_id, resolved_version
        )

    def get_versioned_with_rebuild(
        self,
        namespace: str,
        tenant_id: str,
        skill_id: str,
        version: str,
        source_loader=None,
        rebuild_fn=None,
    ) -> Any:
        value = self.get_versioned(namespace, tenant_id, skill_id, version)
        if value is not None:
            return value

        if source_loader is not None:
            value = source_loader.load_from_source(tenant_id, skill_id, version)
        elif rebuild_fn is not None:
            value = rebuild_fn()
        else:
            raise TypeError("source_loader or rebuild_fn is required")
        self.set_versioned(namespace, tenant_id, skill_id, version, value)
        return value

    def invalidate_versioned(
        self, namespace: str, tenant_id: str, skill_id: str, version: str
    ) -> None:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        self.delete(key)


class PersistentMockRedisAdapter(MockRedisAdapter):
    def __init__(self, filepath: str):
        super().__init__()
        self._filepath = filepath
        self._load()

    def _load(self):
        import os
        if os.path.exists(self._filepath):
            try:
                with open(self._filepath, "r") as f:
                    self._store = json.load(f)
            except Exception as e:
                pass

    def _save(self):
        import os
        try:
            os.makedirs(os.path.dirname(self._filepath), exist_ok=True)
            with open(self._filepath, "w") as f:
                json.dump(self._store, f)
        except Exception as e:
            pass

    def set(self, key: str, value: Any, ex: Optional[int] = None, **kwargs) -> None:
        super().set(key, value, ex, **kwargs)
        self._save()

    def delete(self, key: str) -> bool:
        res = super().delete(key)
        self._save()
        return res

    def rpush(self, key: str, value: Any) -> int:
        res = super().rpush(key, value)
        self._save()
        return res

    def lpush(self, key: str, value: Any) -> int:
        res = super().lpush(key, value)
        self._save()
        return res

    def ltrim(self, key: str, start: int, end: int) -> bool:
        res = super().ltrim(key, start, end)
        self._save()
        return res

    def hset(self, key: str, field: str, value: Any) -> int:
        res = super().hset(key, field, value)
        self._save()
        return res

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        res = super().hsetnx(key, field, value)
        self._save()
        return res

    def hdel(self, key: str, field: str) -> int:
        res = super().hdel(key, field)
        self._save()
        return res

    def clear(self) -> None:
        super().clear()
        self._save()


class RedisAdapter:
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
            return MockRedisAdapter()
        if url.startswith("https://"):
            url = "rediss://" + url[len("https://") :]
        elif url.startswith("http://"):
            url = "redis://" + url[len("http://") :]
        return cls(url, token)

    def set(self, key: str, value: Any, ex: Optional[int] = None, **kwargs) -> None:
        if not self.client:
            return
        serialized = json.dumps(value)
        self.client.set(key, serialized, ex=ex, **kwargs)

    def get(self, key: str) -> Optional[Any]:
        if not self.client:
            return None
        value = self.client.get(key)
        if value:
            return json.loads(value)
        return None

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
        serialized = json.dumps(value)
        return self.client.rpush(key, serialized)

    def lpush(self, key: str, value: Any) -> int:
        if not self.client:
            return 0
        serialized = json.dumps(value)
        return self.client.lpush(key, serialized)

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
        serialized = json.dumps(value)
        return self.client.hset(key, field, serialized)

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        if not self.client:
            return 0
        serialized = json.dumps(value)
        return self.client.hsetnx(key, field, serialized)

    def hget(self, key: str, field: str) -> Optional[Any]:
        if not self.client:
            return None
        value = self.client.hget(key, field)
        if value:
            return json.loads(value)
        return None

    def hdel(self, key: str, field: str) -> int:
        if not self.client:
            return 0
        return self.client.hdel(key, field)

    def hgetall(self, key: str) -> Dict[str, Any]:
        if not self.client:
            return {}
        data = self.client.hgetall(key)
        return {k: json.loads(v) for k, v in data.items()}

    def close(self):
        if self.client:
            self.client.close()
