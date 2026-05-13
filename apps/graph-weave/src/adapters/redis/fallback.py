import threading
import fnmatch
from collections import OrderedDict
from typing import Any, Dict, List, Optional

class FallbackStorage:
    """In-memory LRU storage for when Redis is unavailable."""

    MAX_KEYS = 1000

    def __init__(self):
        self._store: OrderedDict[str, Any] = OrderedDict()
        self._lock = threading.Lock()

    def get(self, key: str) -> Optional[Any]:
        with self._lock:
            if key in self._store:
                self._store.move_to_end(key)
                return self._store[key]
            return None

    def set(self, key: str, value: Any, ex: Optional[int] = None) -> bool:
        with self._lock:
            if len(self._store) >= self.MAX_KEYS:
                oldest = next(iter(self._store))
                del self._store[oldest]

            self._store[key] = value
            if key in self._store:
                self._store.move_to_end(key)
            return True

    def delete(self, key: str) -> bool:
        with self._lock:
            if key in self._store:
                del self._store[key]
                return True
            return False

    def exists(self, key: str) -> bool:
        with self._lock:
            return key in self._store

    def rpush(self, key: str, value: Any) -> int:
        with self._lock:
            if key not in self._store:
                self._store[key] = []
            elif not isinstance(self._store[key], list):
                self._store[key] = [self._store[key]]

            self._store[key].append(value)
            return len(self._store[key])

    def lpush(self, key: str, value: Any) -> int:
        with self._lock:
            if key not in self._store:
                self._store[key] = []
            elif not isinstance(self._store[key], list):
                self._store[key] = [self._store[key]]

            self._store[key].insert(0, value)
            return len(self._store[key])

    def lrange(self, key: str, start: int, end: int) -> List[Any]:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], list):
                return []
            items = self._store[key]
            if end == -1:
                return items[start:]
            return items[start : end + 1]

    def ltrim(self, key: str, start: int, end: int) -> bool:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], list):
                return False
            if end == -1:
                self._store[key] = self._store[key][start:]
            else:
                self._store[key] = self._store[key][start : end + 1]
            return True

    def mget(self, keys: List[str]) -> Dict[str, Any]:
        with self._lock:
            return {k: self._store[k] for k in keys if k in self._store}

    def ttl(self, key: str) -> int:
        with self._lock:
            if key not in self._store:
                return -2
            return -1

    def clear(self):
        with self._lock:
            self._store.clear()

    def hset(self, key: str, field: str, value: Any) -> int:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], dict):
                self._store[key] = {}
            self._store[key][field] = value
            return 1

    def hsetnx(self, key: str, field: str, value: Any) -> int:
        with self._lock:
            if key not in self._store or not isinstance(self._store[key], dict):
                self._store[key] = {}
            if field in self._store[key]:
                return 0
            self._store[key][field] = value
            return 1

    def hget(self, key: str, field: str) -> Optional[Any]:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                return self._store[key].get(field)
            return None

    def hdel(self, key: str, field: str) -> int:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                if field in self._store[key]:
                    del self._store[key][field]
                    return 1
            return 0

    def hgetall(self, key: str) -> Dict[str, Any]:
        with self._lock:
            if key in self._store and isinstance(self._store[key], dict):
                return dict(self._store[key])
            return {}

    def keys(self, pattern: str) -> List[str]:
        with self._lock:
            return [k for k in self._store.keys() if fnmatch.fnmatch(k, pattern)]
