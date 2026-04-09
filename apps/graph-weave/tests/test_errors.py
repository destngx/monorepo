import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestErrorHandling:
    def test_execute_invalid_payload_returns_error(self, client):
        response = client.post("/execute", json={"invalid": "payload"})
        assert response.status_code == 422

    def test_execute_missing_required_field(self, client):
        response = client.post(
            "/execute", json={"tenant_id": "test", "workflow_id": "test"}
        )
        assert response.status_code == 422

    def test_error_response_is_json(self, client):
        response = client.post("/execute", json={})
        assert response.headers.get("content-type") == "application/json"

    def test_status_endpoint_returns_json(self, client):
        response = client.get("/execute/test-run-id/status")
        assert response.status_code == 200
        data = response.json()
        assert isinstance(data, dict)
        assert "run_id" in data

    def test_status_endpoint_has_events_field(self, client):
        response = client.get("/execute/test-run-id/status")
        data = response.json()
        assert "events" in data
        assert isinstance(data["events"], list)
