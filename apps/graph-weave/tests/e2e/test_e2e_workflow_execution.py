import time

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


def wait_for_terminal_status(client, run_id, timeout=2.0):
    deadline = time.monotonic() + timeout
    last_data = None
    while time.monotonic() < deadline:
        response = client.get(f"/execute/{run_id}/status")
        assert response.status_code == 200
        last_data = response.json()
        if last_data.get("status") in ["completed", "failed", "cancelled"]:
            return last_data
        time.sleep(0.01)
    return last_data


class TestWorkflowExecutionE2E:
    def test_complete_workflow_submission_and_execution(
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
        assert data["status"] == "queued"

        final = wait_for_terminal_status(client, data["run_id"])
        assert final is not None
        assert final["status"] in ["completed", "failed"]

    def test_event_log_accumulation(
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

        data = wait_for_terminal_status(client, response.json()["run_id"])
        assert data is not None
        assert len(data.get("events", [])) > 0

    def test_state_propagation(
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

        data = wait_for_terminal_status(client, response.json()["run_id"])
        assert data is not None
        assert isinstance(data.get("final_state"), dict)
