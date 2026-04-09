import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestExecutionEndpoint:
    def test_execute_endpoint_exists(self, client):
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
        payload = {
            "tenant_id": "test-tenant",
            "workflow_id": "test-workflow",
            "input": {"key": "value"},
        }
        response = client.post("/execute", json=payload)
        data = response.json()

        assert data["status"] == "pending"
        assert data["workflow_id"] == "test-workflow"
        assert data["tenant_id"] == "test-tenant"
