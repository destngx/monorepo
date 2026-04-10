import pytest

from src.adapters.cache import MockRedisAdapter


class CheckpointService:
    def __init__(self, cache):
        self.cache = cache

    def save_checkpoint(self, tenant_id, thread_id, workflow_state):
        self.cache.set(
            f"checkpoint:{tenant_id}:{thread_id}:latest",
            workflow_state,
        )

    def load_checkpoint(self, tenant_id, thread_id):
        return self.cache.get(f"checkpoint:{tenant_id}:{thread_id}:latest")

    def clear_checkpoint(self, tenant_id, thread_id):
        self.cache.delete(f"checkpoint:{tenant_id}:{thread_id}:latest")


@pytest.fixture
def checkpoint_service():
    return CheckpointService(MockRedisAdapter())


class TestCheckpointRecovery:
    def test_save_checkpoint_uses_latest_key(self, checkpoint_service):
        checkpoint_service.save_checkpoint(
            "tenant-1",
            "thread-1",
            {"node_id": "node-1", "workflow_state": {"step": 1}},
        )

        assert checkpoint_service.cache.exists("checkpoint:tenant-1:thread-1:latest")

    def test_load_checkpoint_returns_saved_state(self, checkpoint_service):
        workflow_state = {"node_id": "node-1", "workflow_state": {"step": 1}}
        checkpoint_service.save_checkpoint("tenant-1", "thread-1", workflow_state)

        assert (
            checkpoint_service.load_checkpoint("tenant-1", "thread-1") == workflow_state
        )

    def test_clear_checkpoint_deletes_saved_state(self, checkpoint_service):
        checkpoint_service.save_checkpoint("tenant-1", "thread-1", {"step": 1})
        checkpoint_service.clear_checkpoint("tenant-1", "thread-1")

        assert checkpoint_service.load_checkpoint("tenant-1", "thread-1") is None
