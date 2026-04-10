import pytest
from fastapi.testclient import TestClient
from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


class TestExecutionEndpoint:
    def setup_method(self):
        get_workflow_store().clear()

    def _create_workflow(self):
        get_workflow_store().create(
            "test-tenant",
            {
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow",
                "name": "Test Workflow",
                "version": "1.0.0",
                "description": "Test workflow",
                "owner": "test-owner",
                "tags": [],
                "definition": {
                    "nodes": [
                        {"id": "entry", "type": "entry", "config": {}},
                        {"id": "exit", "type": "exit", "config": {}},
                    ],
                    "edges": [{"from": "entry", "to": "exit"}],
                    "entry_point": "entry",
                    "exit_point": "exit",
                },
            },
        )

    def test_execute_endpoint_exists(self, client):
        self._create_workflow()
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow",
                "input": {},
            },
        )
        assert response.status_code == 200

    def test_execute_returns_run_id(self, client):
        self._create_workflow()
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow",
                "input": {},
            },
        )
        data = response.json()
        assert "run_id" in data
        assert isinstance(data["run_id"], str)
        assert len(data["run_id"]) > 0

    def test_execute_returns_thread_id(self, client):
        self._create_workflow()
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow",
                "input": {},
            },
        )
        data = response.json()
        assert "thread_id" in data
        assert isinstance(data["thread_id"], str)

    def test_execute_returns_deterministic_response(self, client):
        self._create_workflow()
        payload = {
            "tenant_id": "test-tenant",
            "workflow_id": "test-workflow",
            "input": {"key": "value"},
        }
        response = client.post("/execute", json=payload)
        data = response.json()

        assert data["status"] == "queued"
        assert data["workflow_id"] == "test-workflow"
        assert data["tenant_id"] == "test-tenant"

    def test_execute_initial_status_is_queued(self, client):
        self._create_workflow()
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow",
                "input": {},
            },
        )
        assert response.json()["status"] == "queued"

    def test_execute_does_not_auto_create_missing_workflow(self, client):
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "missing-workflow",
                "input": {},
            },
        )
        assert response.status_code == 404

    def test_execute_status_reflects_executor_result(self, client):
        from src.main import execution_runs

        execution_runs["run-123"] = {
            "status": "completed",
            "workflow_id": "wf-1",
            "tenant_id": "tenant-1",
            "events": [],
            "final_state": {"done": True},
            "hop_count": 2,
        }

        response = client.get("/execute/run-123/status")
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "completed"
        assert data["final_state"] == {"done": True}

    def test_execute_status_missing_run_returns_404(self, client):
        response = client.get("/execute/missing-run/status")
        assert response.status_code == 404
