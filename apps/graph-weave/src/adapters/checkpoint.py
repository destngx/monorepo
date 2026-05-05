from typing import Dict, Any, Optional, List


class RedisCheckpointStore:
    def __init__(self, redis_client: Any):
        self._redis = redis_client

    def save(self, run_id: str, thread_id: str, checkpoint_data: Any) -> None:
        key = f"checkpoint:{run_id}:{thread_id}"
        self._redis.set(key, checkpoint_data)

    def load(self, run_id: str, thread_id: str) -> Optional[Any]:
        key = f"checkpoint:{run_id}:{thread_id}"
        return self._redis.get(key)

    def delete(self, run_id: str, thread_id: str) -> None:
        key = f"checkpoint:{run_id}:{thread_id}"
        self._redis.delete(key)

    def list_for_run(self, run_id: str) -> List[Any]:
        # Note: NamespacedRedisClient doesn't support keys() or scan() easily
        # For now, we return empty list as list_for_run is mostly used in tests
        return []

    def clear(self) -> None:
        pass

