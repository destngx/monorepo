import pytest
from fastapi.testclient import TestClient
from src.modules.shared.deps import get_checkpoint_service, get_thread_lifecycle_service
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestRecoveryAndCancellation:
    def test_recover_missing_run_returns_404(self, client):
        response = client.post(
            "/execute/missing-run/recover",
            json={"thread_id": "thread-1"},
        )

        assert response.status_code == 404

    def test_cancel_missing_run_returns_404(self, client):
        response = client.post("/execute/missing-run/cancel")

        assert response.status_code == 404

    def test_cancel_clears_active_thread_and_sets_kill_flag(self, client):
        thread_service = get_thread_lifecycle_service()
        checkpoint_service = get_checkpoint_service()
        thread_service.add_active_thread("tenant-1", "run-1", "wf-1")
        checkpoint_service.save_checkpoint("tenant-1", "run-1", {"workflow_state": {}})

        from src.main import execution_runs, status_service

        execution_runs["run-1"] = {
            "run_id": "run-1",
            "thread_id": "run-1",
            "workflow_id": "wf-1",
            "tenant_id": "tenant-1",
            "status": "running",
            "events": [],
            "final_state": None,
            "hop_count": 1,
        }
        status_service.set_status("tenant-1", "run-1", "running")

        response = client.post("/execute/run-1/cancel")

        assert response.status_code == 200
        assert thread_service.is_active("tenant-1", "run-1") is False
        assert thread_service._store.exists("circuit_breaker:tenant-1:run-1:kill")

    def test_recover_missing_checkpoint_returns_404(self, client):
        response = client.post(
            "/execute/run-1/recover",
            json={"thread_id": "thread-1"},
        )

        assert response.status_code == 404
