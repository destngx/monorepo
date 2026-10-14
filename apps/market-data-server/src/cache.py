"""
Two-tier caching for market-data-server.

- L1: bounded in-process LRU with per-entry expiry (always on).
- L2: Upstash Redis over its REST API when configured, otherwise a local
  SQLite file so entries survive restarts. By default the SQLite file is
  wealth-management's cache file, with the same table, so one file holds both
  apps' caches. L2 keys carry CACHE_KEY_PREFIX (default "mds:"), and clearing
  the cache deletes only keys under that prefix, never another app's keys.

Concurrent misses for the same key are coalesced so only one upstream call
is made (single-flight).
"""

import os
import json
import time
import asyncio
import logging
import sqlite3
import tempfile
import threading
from collections import OrderedDict
from pathlib import Path
from typing import Optional, Any, Dict, Callable, Awaitable, Tuple, Union
import httpx
from .ttl_utils import calculate_ttl_until_next_5am, DEFAULT_MAX_TTL

logger = logging.getLogger(__name__)


DEFAULT_KEY_PREFIX = "mds:"


def _dumps(data: Any) -> str:
    # default=str keeps pandas Timestamps and similar values serialisable.
    return json.dumps(data, default=str)


class MemoryCache:
    """Bounded LRU of serialised values with per-entry expiry."""

    def __init__(self, max_entries: int = 2048):
        self.max_entries = max_entries
        self._entries: "OrderedDict[str, Tuple[float, str]]" = OrderedDict()

    def get(self, key: str) -> Optional[str]:
        entry = self._entries.get(key)
        if entry is None:
            return None
        expires_at, value = entry
        if expires_at <= time.monotonic():
            del self._entries[key]
            return None
        self._entries.move_to_end(key)
        return value

    def set(self, key: str, value: str, ttl: int) -> None:
        if ttl <= 0:
            return
        self._entries[key] = (time.monotonic() + ttl, value)
        self._entries.move_to_end(key)
        while len(self._entries) > self.max_entries:
            self._entries.popitem(last=False)

    def delete(self, key: str) -> None:
        self._entries.pop(key, None)

    def clear(self) -> None:
        self._entries.clear()

    def __len__(self) -> int:
        return len(self._entries)


class UpstashRedisClient:
    """
    HTTP client for Upstash Redis REST API.
    Reuses one pooled connection instead of opening a new one per command.
    """

    # Keys deleted per DEL command when clearing the prefix.
    DELETE_BATCH = 500

    def __init__(self, url: str, token: str, prefix: str = DEFAULT_KEY_PREFIX, timeout: float = 5.0):
        self.url = url.rstrip("/")
        self.prefix = _require_prefix(prefix)
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
        }
        self.timeout = timeout
        self._client: Optional[httpx.AsyncClient] = None

    def _http(self) -> httpx.AsyncClient:
        if self._client is None or self._client.is_closed:
            self._client = httpx.AsyncClient(
                base_url=self.url,
                headers=self.headers,
                timeout=self.timeout,
                limits=httpx.Limits(max_connections=20, max_keepalive_connections=10),
            )
        return self._client

    async def pipeline(self, commands: list) -> list:
        response = await self._http().post("/pipeline", json=commands)
        response.raise_for_status()
        return response.json()

    async def get_with_ttl(self, key: str) -> Tuple[Optional[str], int]:
        """Return the value and its remaining TTL in seconds (0 if unknown)."""
        try:
            key = self.prefix + key
            get_result, ttl_result = await self.pipeline([["GET", key], ["TTL", key]])
            value = get_result.get("result")
            ttl = ttl_result.get("result") or 0
            return (value, ttl if ttl > 0 else 0) if value else (None, 0)
        except Exception as e:
            logger.warning(f"Redis GET error for {key}: {e}")
            return None, 0

    async def set(self, key: str, value: str, ex: Optional[int] = None) -> bool:
        """Set value in Redis with optional TTL (in seconds)."""
        key = self.prefix + key
        cmd = ["SET", key, value]
        if ex:
            cmd.extend(["EX", str(ex)])
        try:
            await self.pipeline([cmd])
            return True
        except Exception as e:
            logger.warning(f"Redis SET error for {key}: {e}")
            return False

    async def delete(self, key: str) -> bool:
        """Delete key from Redis."""
        key = self.prefix + key
        try:
            await self.pipeline([["DEL", key]])
            return True
        except Exception as e:
            logger.warning(f"Redis DEL error for {key}: {e}")
            return False

    async def flushall(self) -> bool:
        """Delete every key under the prefix; other apps' keys in the same database stay."""
        pattern = _redis_glob_escape(self.prefix) + "*"
        cursor = "0"
        deleted = 0
        try:
            while True:
                (scan,) = await self.pipeline([["SCAN", cursor, "MATCH", pattern, "COUNT", "1000"]])
                cursor, keys = scan["result"]
                for start in range(0, len(keys), self.DELETE_BATCH):
                    await self.pipeline([["DEL", *keys[start : start + self.DELETE_BATCH]]])
                deleted += len(keys)
                if str(cursor) == "0":
                    break
            logger.info(f"✓ Redis cache cleared ({deleted} keys under {self.prefix!r})")
            return True
        except Exception as e:
            logger.error(f"Redis cache clear error: {e}")
            return False

    async def aclose(self) -> None:
        if self._client is not None:
            await self._client.aclose()
            self._client = None


# wealth-management's default cache file (adapter/config DefaultSQLiteCachePath:
# <system temp>/wealth-management/cache.db), so both apps share one file.
DEFAULT_SQLITE_PATH = Path(tempfile.gettempdir()) / "wealth-management" / "cache.db"


def _require_prefix(prefix: str) -> str:
    # An empty prefix would let a cache clear delete other apps' keys.
    if not prefix:
        raise ValueError("CACHE_KEY_PREFIX must not be empty")
    return prefix


def _redis_glob_escape(text: str) -> str:
    return "".join("\\" + char if char in "*?[]\\" else char for char in text)


class SQLiteCache:
    """
    Durable local L2 used when Upstash is not configured. Same async interface
    as UpstashRedisClient; blocking sqlite3 calls run in a worker thread.

    The table is wealth-management's (apps/wealth-management/adapter/cache/
    sqlite_client.go): expires_at is Unix milliseconds, NULL never expires.
    Keep the two schemas identical. Both apps use WAL and a 5 s busy timeout,
    so they can hold the file open at the same time.
    """

    # Purge expired rows once per this many writes.
    PURGE_EVERY = 500

    SCHEMA = (
        "CREATE TABLE IF NOT EXISTS cache_entries ("
        " key TEXT PRIMARY KEY, value TEXT NOT NULL, expires_at INTEGER"
        ")",
        "CREATE INDEX IF NOT EXISTS cache_entries_expires_at"
        " ON cache_entries (expires_at) WHERE expires_at IS NOT NULL",
    )

    def __init__(self, path: Union[str, Path], prefix: str = DEFAULT_KEY_PREFIX):
        self.path = Path(path)
        self.prefix = _require_prefix(prefix)
        # The file can hold wealth-management's copies of ledger data: keep it private.
        self.path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
        self.path.touch(mode=0o600, exist_ok=True)
        os.chmod(self.path, 0o600)
        self._lock = threading.Lock()
        self._writes = 0
        self._conn = sqlite3.connect(self.path, timeout=5.0, check_same_thread=False, isolation_level=None)
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA synchronous=NORMAL")
        for statement in self.SCHEMA:
            self._conn.execute(statement)
        self._purge()

    @staticmethod
    def _now_ms() -> int:
        return int(time.time() * 1000)

    def _purge(self) -> None:
        # Expired rows are dead for every app, so this purges all prefixes.
        with self._lock:
            self._conn.execute(
                "DELETE FROM cache_entries WHERE expires_at IS NOT NULL AND expires_at <= ?",
                (self._now_ms(),),
            )

    def _get(self, key: str) -> Tuple[Optional[str], int]:
        with self._lock:
            row = self._conn.execute(
                "SELECT value, expires_at FROM cache_entries WHERE key = ?", (self.prefix + key,)
            ).fetchone()
        if row is None:
            return None, 0
        value, expires_at = row
        if expires_at is None:
            return value, 0
        remaining = (expires_at - self._now_ms()) // 1000
        return (value, remaining) if remaining > 0 else (None, 0)

    def _set(self, key: str, value: str, ex: int) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT INTO cache_entries (key, value, expires_at) VALUES (?, ?, ?)"
                " ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at",
                (self.prefix + key, value, self._now_ms() + ex * 1000),
            )
            self._writes += 1
            due = self._writes % self.PURGE_EVERY == 0
        if due:
            self._purge()

    def _execute(self, sql: str, params: tuple = ()) -> None:
        with self._lock:
            self._conn.execute(sql, params)

    # Matches this app's rows; its two arguments are the prefix length and the prefix.
    _OWN_ROWS = "substr(key, 1, ?) = ?"

    async def get_with_ttl(self, key: str) -> Tuple[Optional[str], int]:
        try:
            return await asyncio.to_thread(self._get, key)
        except sqlite3.Error as e:
            logger.warning(f"SQLite cache GET error for {key}: {e}")
            return None, 0

    async def set(self, key: str, value: str, ex: Optional[int] = None) -> bool:
        if not ex or ex <= 0:
            return False
        try:
            await asyncio.to_thread(self._set, key, value, ex)
            return True
        except sqlite3.Error as e:
            logger.warning(f"SQLite cache SET error for {key}: {e}")
            return False

    async def delete(self, key: str) -> bool:
        try:
            await asyncio.to_thread(self._execute, "DELETE FROM cache_entries WHERE key = ?", (self.prefix + key,))
            return True
        except sqlite3.Error as e:
            logger.warning(f"SQLite cache DEL error for {key}: {e}")
            return False

    async def flushall(self) -> bool:
        """Delete every row under the prefix; other apps' rows in the file stay."""
        try:
            await asyncio.to_thread(
                self._execute,
                f"DELETE FROM cache_entries WHERE {self._OWN_ROWS}",
                (len(self.prefix), self.prefix),
            )
            return True
        except sqlite3.Error as e:
            logger.error(f"SQLite cache flush error: {e}")
            return False

    def __len__(self) -> int:
        """Live rows under the prefix."""
        with self._lock:
            return self._conn.execute(
                f"SELECT COUNT(*) FROM cache_entries WHERE {self._OWN_ROWS}"
                " AND (expires_at IS NULL OR expires_at > ?)",
                (len(self.prefix), self.prefix, self._now_ms()),
            ).fetchone()[0]

    async def aclose(self) -> None:
        with self._lock:
            self._conn.close()


class CacheManager:
    """
    Manages caching for upstream API responses.
    Reads go L1 (memory) -> L2 (Upstash or SQLite) -> upstream; writes go to both tiers.
    """

    def __init__(
        self,
        redis_url: Optional[str] = None,
        redis_token: Optional[str] = None,
        memory_max_entries: Optional[int] = None,
        sqlite_path: Optional[Union[str, Path]] = None,
        enabled: Optional[bool] = None,
        key_prefix: Optional[str] = None,
    ):
        self.redis_url = redis_url if redis_url is not None else os.getenv("UPSTASH_REDIS_REST_URL")
        self.redis_token = redis_token if redis_token is not None else os.getenv("UPSTASH_REDIS_REST_TOKEN")
        self.memory = MemoryCache(
            memory_max_entries or int(os.getenv("CACHE_MEMORY_MAX_ENTRIES", "2048"))
        )
        self._inflight: Dict[str, asyncio.Task] = {}
        l2_enabled = CacheConfig.CACHE_ENABLED if enabled is None else enabled
        self.key_prefix = _require_prefix(
            key_prefix if key_prefix is not None else os.getenv("CACHE_KEY_PREFIX") or DEFAULT_KEY_PREFIX
        )

        self.client: Optional[Union[UpstashRedisClient, SQLiteCache]] = None
        self.backend = "memory"
        if l2_enabled and self.redis_url and self.redis_token:
            self.client = UpstashRedisClient(self.redis_url, self.redis_token, self.key_prefix)
            self.backend = "memory + Upstash Redis"
        elif l2_enabled:
            path = sqlite_path or os.getenv("CACHE_SQLITE_PATH") or DEFAULT_SQLITE_PATH
            self.client = SQLiteCache(path, self.key_prefix)
            self.backend = f"memory + SQLite ({path})"
        logger.info(f"✓ Cache layer initialized ({self.backend}, key prefix {self.key_prefix!r})")

    @property
    def enabled(self) -> bool:
        return True

    @staticmethod
    def _make_key(endpoint: str, params: Dict[str, Any]) -> str:
        """Generate consistent cache key from endpoint and parameters; L2 clients add the prefix."""
        param_str = ":".join(f"{k}={v}" for k, v in sorted(params.items()))
        return f"{endpoint}:{param_str}"

    async def _read(self, key: str) -> Optional[str]:
        raw = self.memory.get(key)
        if raw is not None:
            return raw
        if self.client is None:
            return None
        raw, remaining_ttl = await self.client.get_with_ttl(key)
        if raw is not None and remaining_ttl:
            self.memory.set(key, raw, remaining_ttl)
        return raw

    async def _write(self, key: str, raw: str, ttl: int) -> None:
        self.memory.set(key, raw, ttl)
        if self.client is not None:
            await self.client.set(key, raw, ex=ttl)

    async def get(self, endpoint: str, params: Dict[str, Any]) -> Optional[Any]:
        """Get cached data if available."""
        key = self._make_key(endpoint, params)
        try:
            raw = await self._read(key)
            return json.loads(raw) if raw is not None else None
        except Exception as e:
            logger.warning(f"Cache retrieval error: {e}")
            return None

    async def set(
        self,
        endpoint: str,
        params: Dict[str, Any],
        data: Any,
        ttl: Optional[int] = None,
    ) -> bool:
        """Set cached data, defaulting the TTL to the next 5am GMT+7 reset."""
        key = self._make_key(endpoint, params)
        if ttl is None:
            ttl = calculate_ttl_until_next_5am(max_ttl=DEFAULT_MAX_TTL)
        try:
            await self._write(key, _dumps(data), ttl)
            return True
        except Exception as e:
            logger.warning(f"Cache storage error: {e}")
            return False

    async def get_or_compute(
        self,
        endpoint: str,
        params: Dict[str, Any],
        compute: Callable[[], Awaitable[Any]],
        ttl: int,
        should_cache: Callable[[Any], bool],
        refresh: bool = False,
    ) -> Tuple[Any, bool]:
        """
        Return (data, was_cached). On a miss, concurrent callers for the same
        key share a single `compute()` call; failures are not cached.
        `refresh` skips the read but still caches the fresh answer.
        """
        if not refresh:
            cached = await self.get(endpoint, params)
            if cached is not None:
                return cached, True

        key = self._make_key(endpoint, params)
        task = self._inflight.get(key)
        if task is None:

            async def run() -> Any:
                try:
                    data = await compute()
                    if should_cache(data):
                        await self.set(endpoint, params, data, ttl=ttl)
                    return data
                finally:
                    self._inflight.pop(key, None)

            task = asyncio.create_task(run())
            self._inflight[key] = task

        # Shield so one disconnecting client does not cancel the shared fetch.
        data = await asyncio.shield(task)
        # Callers get their own copy since followers share the leader's object.
        return json.loads(_dumps(data)) if isinstance(data, (dict, list)) else data, False

    async def invalidate(
        self, endpoint: str, params: Optional[Dict[str, Any]] = None
    ) -> bool:
        """Invalidate cache for a specific endpoint and parameters."""
        if not params:
            return True
        key = self._make_key(endpoint, params)
        self.memory.delete(key)
        if self.client is not None:
            return await self.client.delete(key)
        return True

    async def flush_all(self) -> bool:
        """Clear the in-memory cache and this app's keys in the L2 tier."""
        self.memory.clear()
        if self.client is None:
            return True
        return await self.client.flushall()

    async def aclose(self) -> None:
        if self.client is not None:
            await self.client.aclose()


TTL = Union[int, Callable[[Dict[str, Any]], int]]


class CacheConfig:
    """
    Cache TTL policy (seconds). Every TTL is also capped at the next 5am GMT+7
    reset so no entry survives into a new trading day.
    """

    MAX_TTL = DEFAULT_MAX_TTL

    QUOTE_TTL = 60
    INTRADAY_TTL = 60
    PRICE_BOARD_TTL = 60
    LISTING_TTL = 3600
    # Ranges that include today still change during the session.
    HISTORICAL_LIVE_TTL = 300
    # None means "until the next 5am GMT+7 reset".
    HISTORICAL_CLOSED_TTL = None
    FINANCE_TTL = None
    COMPANY_TTL = None

    # false keeps only the in-process tier.
    CACHE_ENABLED = os.getenv("CACHE_ENABLED", "true").lower() == "true"

    @classmethod
    def get_ttl(cls, ttl: Optional[int] = None) -> int:
        """Resolve a policy TTL, capped at the next 5am GMT+7 reset."""
        until_reset = calculate_ttl_until_next_5am(max_ttl=cls.MAX_TTL)
        return until_reset if ttl is None else min(ttl, until_reset)

    @classmethod
    def describe(cls) -> Dict[str, Any]:
        def fmt(ttl: Optional[int]) -> Any:
            return "until 5am GMT+7" if ttl is None else ttl

        return {
            "quote": fmt(cls.QUOTE_TTL),
            "intraday": fmt(cls.INTRADAY_TTL),
            "price_board": fmt(cls.PRICE_BOARD_TTL),
            "listing": fmt(cls.LISTING_TTL),
            "historical_open_range": fmt(cls.HISTORICAL_LIVE_TTL),
            "historical_closed_range": fmt(cls.HISTORICAL_CLOSED_TTL),
            "finance": fmt(cls.FINANCE_TTL),
            "company": fmt(cls.COMPANY_TTL),
        }
