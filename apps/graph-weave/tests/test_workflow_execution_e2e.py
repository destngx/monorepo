import pytest
from fastapi.testclient import TestClient
from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def setup_test_workflow(workflow_multi_node, test_tenant_id):
    store = get_workflow_store()
    store.create(test_tenant_id, workflow_multi_node)
    return workflow_multi_node


class TestPostExecuteEndpoint:
    def test_execute_with_valid_workflow(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert "run_id" in data
        assert "status" in data
        assert data["workflow_id"] == test_workflow_id
        assert data["tenant_id"] == test_tenant_id
        assert data["status"] in ["completed", "pending", "running", "error"]

    def test_execute_with_missing_workflow(
        self, client, test_tenant_id, test_input_data
    ):
        response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": "nonexistent:v1.0.0",
                "input": test_input_data,
            },
        )

        assert response.status_code in [404, 200]

    def test_execute_returns_run_id(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        assert response.status_code == 200
        data = response.json()
        run_id = data.get("run_id")
        assert run_id is not None
        assert isinstance(run_id, str)
        assert len(run_id) > 0

    def test_execute_requires_input(
        self, client, setup_test_workflow, test_tenant_id, test_workflow_id
    ):
        response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
            },
        )

        assert response.status_code in [400, 422]


class TestGetStatusEndpoint:
    def test_get_status_after_execute(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        execute_response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        run_id = execute_response.json()["run_id"]

        status_response = client.get(f"/execute/{run_id}/status")

        assert status_response.status_code == 200
        data = status_response.json()
        assert "run_id" in data
        assert data["run_id"] == run_id
        assert "status" in data
        assert "events" in data

    def test_get_status_returns_events(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        execute_response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        run_id = execute_response.json()["run_id"]

        status_response = client.get(f"/execute/{run_id}/status")

        data = status_response.json()
        events = data.get("events", [])
        assert isinstance(events, list)
        assert len(events) > 0

    def test_get_status_includes_final_state(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        execute_response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        run_id = execute_response.json()["run_id"]

        status_response = client.get(f"/execute/{run_id}/status")

        data = status_response.json()
        assert "final_state" in data
        final_state = data["final_state"]
        assert isinstance(final_state, dict)

    def test_get_status_with_invalid_run_id(self, client):
        response = client.get("/execute/invalid-run-id/status")

        assert response.status_code in [200, 404]

    def test_get_status_events_have_timestamps(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        execute_response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        run_id = execute_response.json()["run_id"]

        status_response = client.get(f"/execute/{run_id}/status")

        data = status_response.json()
        events = data.get("events", [])

        for event in events:
            assert "timestamp" in event
            assert event["timestamp"].endswith("Z")


class TestExecuteStatusFlow:
    def test_complete_flow_post_then_get(
        self,
        client,
        setup_test_workflow,
        test_tenant_id,
        test_workflow_id,
        test_input_data,
    ):
        execute_response = client.post(
            "/execute",
            json={
                "tenant_id": test_tenant_id,
                "workflow_id": test_workflow_id,
                "input": test_input_data,
            },
        )

        assert execute_response.status_code == 200
        execute_data = execute_response.json()

        run_id = execute_data["run_id"]
        assert run_id is not None

        status_response = client.get(f"/execute/{run_id}/status")
        assert status_response.status_code == 200
        status_data = status_response.json()

        assert status_data["run_id"] == run_id
        assert "status" in status_data
        assert "events" in status_data
        assert len(status_data["events"]) > 0
