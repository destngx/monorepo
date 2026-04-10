import pytest

from src.adapters.cache import MockRedisAdapter


class ThreadLifecycleService:
    def __init__(self, cache):
        self.cache = cache

    def add_active_thread(self, tenant_id, thread_id, workflow_id):
        self.cache.set(
            f"active_threads:{tenant_id}:{thread_id}",
            {
                "thread_id": thread_id,
                "workflow_id": workflow_id,
                "tenant_id": tenant_id,
            },
        )

    def is_active(self, tenant_id, thread_id):
        return self.cache.exists(f"active_threads:{tenant_id}:{thread_id}")

    def remove_active_thread(self, tenant_id, thread_id):
        self.cache.delete(f"active_threads:{tenant_id}:{thread_id}")

    def set_kill_flag(self, tenant_id, thread_id):
        self.cache.set(f"circuit_breaker:{tenant_id}:{thread_id}:kill", True)


@pytest.fixture
def thread_service():
    return ThreadLifecycleService(MockRedisAdapter())


class TestThreadLifecycle:
    def test_add_active_thread_marks_active(self, thread_service):
        thread_service.add_active_thread("tenant-1", "thread-1", "wf-1")

        assert thread_service.is_active("tenant-1", "thread-1") is True

    def test_remove_active_thread_clears_membership(self, thread_service):
        thread_service.add_active_thread("tenant-1", "thread-1", "wf-1")
        thread_service.remove_active_thread("tenant-1", "thread-1")

        assert thread_service.is_active("tenant-1", "thread-1") is False

    def test_set_kill_flag_records_flag(self, thread_service):
        thread_service.set_kill_flag("tenant-1", "thread-1")

        assert thread_service.cache.exists("circuit_breaker:tenant-1:thread-1:kill")
