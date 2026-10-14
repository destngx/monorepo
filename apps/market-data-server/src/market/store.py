"""
Durable SQLite store for last-good copies (MDS-D5). When every provider in
a chain fails, the service serves the newest copy of the answer instead of
an error and marks it stale.
"""

import asyncio
import json
import os
import sqlite3
import threading
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Optional, Union

DEFAULT_DB_PATH = Path(__file__).resolve().parents[2] / "data" / "market.sqlite3"

# Long enough to ride out an outage over a weekend.
LAST_GOOD_TTL_SECONDS = 7 * 24 * 3600


@dataclass
class LastGood:
    record: Any
    as_of: str


class LastGoodStore:
    def __init__(self, path: Union[str, Path, None] = None, ttl_seconds: int = LAST_GOOD_TTL_SECONDS):
        target = str(path or os.getenv("MARKET_DB_PATH") or DEFAULT_DB_PATH)
        if target != ":memory:":
            Path(target).parent.mkdir(parents=True, exist_ok=True)
        self.path = target
        self.ttl_seconds = ttl_seconds
        self._lock = threading.Lock()
        self._conn = sqlite3.connect(target, check_same_thread=False, isolation_level=None)
        self._conn.execute("PRAGMA journal_mode=WAL")
        self._conn.execute("PRAGMA synchronous=NORMAL")
        self._conn.execute(
            "CREATE TABLE IF NOT EXISTS last_good ("
            " key TEXT PRIMARY KEY, record TEXT NOT NULL, as_of TEXT NOT NULL, expires_at REAL NOT NULL"
            ") WITHOUT ROWID"
        )
        with self._lock:
            self._conn.execute("DELETE FROM last_good WHERE expires_at <= ?", (time.time(),))

    def _put(self, key: str, record: str, as_of: str) -> None:
        with self._lock:
            self._conn.execute(
                "INSERT INTO last_good (key, record, as_of, expires_at) VALUES (?, ?, ?, ?)"
                " ON CONFLICT(key) DO UPDATE SET record = excluded.record, as_of = excluded.as_of, expires_at = excluded.expires_at",
                (key, record, as_of, time.time() + self.ttl_seconds),
            )

    def _get(self, key: str) -> Optional[LastGood]:
        with self._lock:
            row = self._conn.execute("SELECT record, as_of, expires_at FROM last_good WHERE key = ?", (key,)).fetchone()
        if row is None or row[2] <= time.time():
            return None
        return LastGood(record=json.loads(row[0]), as_of=row[1])

    async def put(self, key: str, record: Any, as_of: str) -> None:
        await asyncio.to_thread(self._put, key, json.dumps(record), as_of)

    async def get(self, key: str) -> Optional[LastGood]:
        return await asyncio.to_thread(self._get, key)

    async def aclose(self) -> None:
        with self._lock:
            self._conn.close()
