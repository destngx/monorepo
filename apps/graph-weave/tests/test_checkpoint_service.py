from src.adapters.cache import MockRedisAdapter
from src.services.checkpoint_service import CheckpointService


def test_checkpoint_service_roundtrip():
    cache = MockRedisAdapter()
    service = CheckpointService(cache)
    state = {"node_id": "node-1", "workflow_state": {"step": 1}}

    service.save_checkpoint("tenant-1", "thread-1", state)

    assert service.load_checkpoint("tenant-1", "thread-1") == {
        "tenant_id": "tenant-1",
        "thread_id": "thread-1",
        "workflow_state": state,
    }
