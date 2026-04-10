import pytest

from src.models import StatusEnum
from src.adapters.cache import MockRedisAdapter
from src.services.status_service import StatusService


class TestStatusLifecycle:
    def test_status_enum_has_seven_states(self):
        assert [s.value for s in StatusEnum] == [
            "queued",
            "validating",
            "pending",
            "running",
            "completed",
            "failed",
            "cancelled",
        ]

    def test_set_and_get_status(self):
        service = StatusService()
        service.set_status(
            "tenant-1", "run-1", StatusEnum.queued, {"workflow_id": "wf-1"}
        )

        record = service.get_status("tenant-1", "run-1")

        assert record is not None
        assert record["status"] == "queued"
        assert record["workflow_id"] == "wf-1"

    def test_transition_updates_status(self):
        service = StatusService()
        service.set_status("tenant-1", "run-1", StatusEnum.queued)

        updated = service.transition_status("tenant-1", "run-1", StatusEnum.validating)

        assert updated["status"] == "validating"

    def test_transition_accepts_string_status(self):
        service = StatusService()
        service.set_status("tenant-1", "run-1", StatusEnum.queued)

        updated = service.transition_status("tenant-1", "run-1", "running")

        assert updated["status"] == "running"

    def test_invalid_status_raises_value_error(self):
        service = StatusService()

        with pytest.raises(ValueError):
            service.set_status("tenant-1", "run-1", "invalid-status")

    def test_set_status_persists_to_status_key(self):
        cache = MockRedisAdapter()
        service = StatusService(cache)

        service.set_status("tenant-1", "run-1", StatusEnum.queued)

        assert cache.exists("run:tenant-1:run-1:status")

    def test_get_status_reads_from_cache(self):
        cache = MockRedisAdapter()
        service = StatusService(cache)
        cache.set(
            "run:tenant-1:run-1:status",
            {
                "tenant_id": "tenant-1",
                "run_id": "run-1",
                "status": "running",
                "events": [],
                "final_state": None,
                "hop_count": 1,
                "workflow_id": "wf-1",
            },
        )

        record = service.get_status("tenant-1", "run-1")

        assert record is not None
        assert record["status"] == "running"

    def test_status_persists_across_service_instances(self):
        cache = MockRedisAdapter()
        service1 = StatusService(cache)
        service1.set_status("tenant-1", "run-1", StatusEnum.queued)

        service2 = StatusService(cache)
        record = service2.get_status("tenant-1", "run-1")

        assert record is not None
        assert record["status"] == "queued"

    def test_transition_emits_shared_event_log(self):
        cache = MockRedisAdapter()
        service = StatusService(cache)
        service.set_status("tenant-1", "run-1", StatusEnum.queued)

        service.transition_status("tenant-1", "run-1", StatusEnum.running)

        events = cache.get("run:tenant-1:run-1:events")
        assert isinstance(events, list)
        assert events[-1]["type"] == "status.changed"
