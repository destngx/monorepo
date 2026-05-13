import json
import os
import fnmatch
from typing import Optional, Dict, Any, List

class MockRedisAdapter:
    """In-memory Redis implementation for testing."""
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

    def close(self) -> None:
        pass

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

    def keys(self, pattern: str) -> List[str]:
        return [k for k in self._store.keys() if fnmatch.fnmatch(k, pattern)]

    def _build_versioned_key(
        self, namespace: str, tenant_id: str, skill_id: str, version: str
    ) -> str:
        return f"{namespace}:{tenant_id}:{skill_id}:{version}"

    def set_versioned(
        self, namespace: str, tenant_id: str, skill_id: str, version: str, value: Any
    ) -> None:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        self.set(key, value)

    def get_versioned(
        self, namespace: str, tenant_id: str, skill_id: str, version: str
    ) -> Optional[Any]:
        key = self._build_versioned_key(namespace, tenant_id, skill_id, version)
        return self.get(key)

    def get_versioned_with_fallback(
        self, namespace: str, tenant_id: str, skill_id: str, version: Optional[str]
    ) -> Optional[Any]:
        resolved_version = version or "latest"
        return self.get_versioned(namespace, tenant_id, skill_id, resolved_version)

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
    """File-backed in-memory Redis implementation."""
    def __init__(self, filepath: str):
        super().__init__()
        self._filepath = filepath
        self._load()

    def _load(self):
        if os.path.exists(self._filepath):
            try:
                with open(self._filepath, "r") as f:
                    self._store = json.load(f)
            except:
                pass

    def _save(self):
        try:
            os.makedirs(os.path.dirname(self._filepath), exist_ok=True)
            with open(self._filepath, "w") as f:
                json.dump(self._store, f)
        except:
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
