import pytest
import json
from src.adapters.checkpoint import RedisCheckpointStore
from src.adapters.cache import MockRedisAdapter
from src.adapters.redis_circuit_breaker import NamespacedRedisClient, FallbackStorage


@pytest.fixture
def checkpoint_store():
    cache = MockRedisAdapter()
    client = NamespacedRedisClient(cache, FallbackStorage())
    return RedisCheckpointStore(client)


class TestCheckpointStorage:
    def test_save_checkpoint(self, checkpoint_store):
        checkpoint_data = {"node": "start", "state": {"value": 1}}
        checkpoint_store.save("run-1", "thread-1", checkpoint_data)

    def test_load_checkpoint(self, checkpoint_store):
        checkpoint_data = {"node": "start", "state": {"value": 1}}
        checkpoint_store.save("run-1", "thread-1", checkpoint_data)

        loaded = checkpoint_store.load("run-1", "thread-1")
        assert loaded == checkpoint_data

    def test_load_nonexistent_checkpoint(self, checkpoint_store):
        loaded = checkpoint_store.load("nonexistent", "thread")
        assert loaded is None

    def test_list_checkpoints(self, checkpoint_store):
        checkpoint_store.save("run-1", "thread-1", {"node": "start"})
        checkpoint_store.save("run-2", "thread-2", {"node": "middle"})

        checkpoints = checkpoint_store.list_for_run("run-1")
        assert len(checkpoints) == 1

    def test_checkpoint_isolation_by_thread(self, checkpoint_store):
        checkpoint_store.save("run-1", "thread-1", {"data": "thread1"})
        checkpoint_store.save("run-1", "thread-2", {"data": "thread2"})

        assert checkpoint_store.load("run-1", "thread-1") == {"data": "thread1"}
        assert checkpoint_store.load("run-1", "thread-2") == {"data": "thread2"}

    def test_delete_checkpoint(self, checkpoint_store):
        checkpoint_store.save("run-1", "thread-1", {"data": "test"})
        checkpoint_store.delete("run-1", "thread-1")

        loaded = checkpoint_store.load("run-1", "thread-1")
        assert loaded is None

    def test_checkpoint_key_shape_matches_thread_contract(self, checkpoint_store):
        checkpoint_store.save("run-1", "thread-1", {"node": "start"})
        assert checkpoint_store.load("run-1", "thread-1") is not None
