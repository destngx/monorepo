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
                # Use print or a simple logger if available
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

    def delete(self, key: str) -> None:
        super().delete(key)
        self._save()

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
        if res == 1:
            self._save()
        return res

    def hdel(self, key: str, field: str) -> int:
        res = super().hdel(key, field)
        if res == 1:
            self._save()
        return res

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

    def hset(self, key: str, field: str, value: Any) -> int:
        super().hset(key, field, value)
        if hasattr(self._client, "hset"):
            serialized = json.dumps(value) if not isinstance(value, (str, int, float)) or isinstance(value, bool) else value
            return self._client.hset(key, field, serialized)
        return 1

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        res = super().hsetnx(key, field, value)
        if res == 0:
            return 0
        if hasattr(self._client, "hsetnx"):
            serialized = json.dumps(value) if not isinstance(value, (str, int, float)) or isinstance(value, bool) else value
            return self._client.hsetnx(key, field, serialized)
        return 1

    def hget(self, key: str, field: str) -> Optional[Any]:
        if hasattr(self._client, "hget"):
            value = self._client.hget(key, field)
            if value is not None:
                try:
                    return json.loads(value)
                except (json.JSONDecodeError, TypeError):
                    return value
        return super().hget(key, field)

    def hdel(self, key: str, field: str) -> int:
        super().hdel(key, field)
        if hasattr(self._client, "hdel"):
            return self._client.hdel(key, field)
        return 0

    def hgetall(self, key: str) -> Dict[str, Any]:
        if hasattr(self._client, "hgetall"):
            items = self._client.hgetall(key)
            result = {}
            for k, v in items.items():
                try:
                    result[k] = json.loads(v)
                except (json.JSONDecodeError, TypeError):
                    result[k] = v
            return result
        return super().hgetall(key)
