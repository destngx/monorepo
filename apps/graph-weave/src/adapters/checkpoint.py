from typing import Dict, Any, Optional, List


class RedisCheckpointStore:
    def __init__(self, redis_client: Any):
        self._redis = redis_client

    def save(self, run_id: str, thread_id: str, checkpoint_data: Any) -> None:
        key = f"checkpoint:{run_id}:{thread_id}"
        self._redis.set(key, checkpoint_data)
        # Track thread IDs for this run to support listing
        self._redis.hset(f"run_threads:{run_id}", thread_id, "1")

    def load(self, run_id: str, thread_id: str) -> Optional[Any]:
        key = f"checkpoint:{run_id}:{thread_id}"
        return self._redis.get(key)

    def delete(self, run_id: str, thread_id: str) -> None:
        key = f"checkpoint:{run_id}:{thread_id}"
        self._redis.delete(key)
        self._redis.hdel(f"run_threads:{run_id}", thread_id)

    def list_for_run(self, run_id: str) -> List[Any]:
        # NamespacedRedisClient doesn't support keys() or scan()
        # We use a hash to track thread IDs for each run_id
        thread_ids = self._redis.hgetall(f"run_threads:{run_id}").keys()
        checkpoints = []
        for tid in thread_ids:
            cp = self.load(run_id, tid)
            if cp:
                checkpoints.append(cp)
        return checkpoints

    def clear(self) -> None:
        # Note: This is mainly for tests with MockRedisAdapter
        if hasattr(self._redis, "redis_client") and hasattr(self._redis.redis_client, "clear"):
            self._redis.redis_client.clear()
        elif hasattr(self._redis, "clear"):
            self._redis.clear()
