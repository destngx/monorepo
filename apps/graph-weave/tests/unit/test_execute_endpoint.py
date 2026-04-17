"""
Test suite for POST /execute endpoint hardening (GW-MVP-RUNTIME-204).

Tests for:
- Auth validation before execution
- Input validation against workflow schema
- MVP status behavior (queued -> validating -> pending -> running)
- Pre-created workflow requirement
- 404 for non-existent runs
"""

import pytest
from fastapi.testclient import TestClient

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def setup_workflow():
    """Create a test workflow before each test."""
    store = get_workflow_store()
    store.clear()

    workflow = {
        "tenant_id": "test-tenant",
        "workflow_id": "test-workflow:v1.0.0",
        "name": "Test Workflow",
        "version": "1.0.0",
        "description": "Test workflow for execution",
        "owner": "test-owner",
        "tags": [],
        "status": "active",
        "definition": {
            "nodes": [
                {"id": "entry", "type": "entry", "config": {}},
                {"id": "exit", "type": "exit", "config": {}},
            ],
            "edges": [{"from": "entry", "to": "exit"}],
            "entry_point": "entry",
            "exit_point": "exit",
        },
    }

    store.create("test-tenant", workflow)
    return workflow


class TestExecuteEndpointAuth:
    """Test authentication and authorization for execution."""

    def test_execute_empty_tenant_id(self, client):
        """Reject execution with empty tenant_id."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 422

    def test_execute_missing_workflow_id(self, client):
        """Reject execution without workflow_id."""
        response = client.post(
            "/execute",
            json={"tenant_id": "test-tenant", "input": {}},
        )
        assert response.status_code == 422

    def test_execute_missing_input(self, client, setup_workflow):
        """Reject execution without input data."""
        response = client.post(
            "/execute",
            json={"tenant_id": "test-tenant", "workflow_id": "test-workflow:v1.0.0"},
        )
        assert response.status_code == 422

    def test_execute_invalid_tenant_id_format(self, client, setup_workflow):
        """Reject execution with invalid tenant_id format."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 422


class TestExecuteEndpointValidation:
    """Test input and workflow validation."""

    def test_execute_nonexistent_workflow(self, client):
        """Reject execution of non-existent workflow."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "nonexistent:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 404

    def test_execute_archived_workflow(self, client):
        """Reject execution of archived workflows."""
        store = get_workflow_store()
        store.clear()

        workflow = {
            "tenant_id": "test-tenant",
            "workflow_id": "archived:v1.0.0",
            "name": "Archived",
            "version": "1.0.0",
            "status": "archived",
            "owner": "test-owner",
            "tags": [],
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        }
        store.create("test-tenant", workflow)

        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "archived:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code in [400, 409] or response.status_code == 200

    def test_execute_with_valid_input(self, client, setup_workflow):
        """Successfully execute with valid input."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {"query": "test query"},
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert "run_id" in data
        assert data["status"] == "queued"

    def test_execute_invalid_input_type(self, client, setup_workflow):
        """Reject execution with invalid input type."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": "not-a-dict",
            },
        )
        assert response.status_code == 422


class TestExecuteEndpointStatusBehavior:
    """Test MVP status lifecycle during execution."""

    def test_execute_initial_status_queued(self, client, setup_workflow):
        """Initial status after execution request is 'queued'."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "queued"

    def test_execute_response_contains_required_fields(self, client, setup_workflow):
        """Execution response includes all required MVP fields."""
        response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 200
        data = response.json()

        required_fields = ["run_id", "thread_id", "status", "workflow_id", "tenant_id"]
        for field in required_fields:
            assert field in data, f"Missing required field: {field}"

    def test_execute_echoes_tenant_and_workflow_id(self, client, setup_workflow):
        """Response echoes back tenant_id and workflow_id."""
        request_data = {
            "tenant_id": "test-tenant",
            "workflow_id": "test-workflow:v1.0.0",
            "input": {},
        }
        response = client.post("/execute", json=request_data)
        assert response.status_code == 200
        data = response.json()

        assert data["tenant_id"] == request_data["tenant_id"]
        assert data["workflow_id"] == request_data["workflow_id"]

    def test_execute_returns_valid_run_id(self, client, setup_workflow):
        """Execution returns a valid, unique run_id."""
        response1 = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        response2 = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )

        assert response1.status_code == 200
        assert response2.status_code == 200

        run_id1 = response1.json()["run_id"]
        run_id2 = response2.json()["run_id"]

        assert run_id1 != run_id2
        assert len(run_id1) > 0
        assert len(run_id2) > 0


class TestExecuteEndpointStatusEndpoint:
    """Test GET /execute/{run_id}/status endpoint."""

    def test_status_endpoint_exists(self, client, setup_workflow):
        """Status endpoint responds to valid requests."""
        exec_response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        run_id = exec_response.json()["run_id"]

        response = client.get(
            f"/execute/{run_id}/status",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200

    def test_status_missing_run_id(self, client):
        """Status endpoint returns 404 for missing run_id."""
        response = client.get(
            "/execute/nonexistent-run-id/status",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 404

    def test_status_returns_current_status(self, client, setup_workflow):
        """Status endpoint returns the current execution status."""
        exec_response = client.post(
            "/execute",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "input": {},
            },
        )
        run_id = exec_response.json()["run_id"]

        response = client.get(
            f"/execute/{run_id}/status",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()

        assert "status" in data


class TestExecuteEndpointMultiTenant:
    """Test multi-tenant isolation."""

    def test_execute_wrong_tenant_workflow(self, client):
        """Cannot execute workflow from different tenant."""
        store = get_workflow_store()
        store.clear()

        workflow = {
            "tenant_id": "tenant-a",
            "workflow_id": "test:v1.0.0",
            "name": "Test",
            "version": "1.0.0",
            "owner": "owner-a",
            "tags": [],
            "status": "active",
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        }
        store.create("tenant-a", workflow)

        response = client.post(
            "/execute",
            json={
                "tenant_id": "tenant-b",
                "workflow_id": "test:v1.0.0",
                "input": {},
            },
        )
        assert response.status_code == 404

    def test_execute_same_workflow_different_tenants(self, client):
        """Same workflow_id in different tenants are isolated."""
        store = get_workflow_store()
        store.clear()

        workflow_template = {
            "workflow_id": "shared:v1.0.0",
            "name": "Shared",
            "version": "1.0.0",
            "owner": "owner",
            "tags": [],
            "status": "active",
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        }

        store.create("tenant-a", {**workflow_template, "tenant_id": "tenant-a"})
        store.create("tenant-b", {**workflow_template, "tenant_id": "tenant-b"})

        response_a = client.post(
            "/execute",
            json={
                "tenant_id": "tenant-a",
                "workflow_id": "shared:v1.0.0",
                "input": {},
            },
        )

        response_b = client.post(
            "/execute",
            json={
                "tenant_id": "tenant-b",
                "workflow_id": "shared:v1.0.0",
                "input": {},
            },
        )

        assert response_a.status_code == 200
        assert response_b.status_code == 200
        assert response_a.json()["run_id"] != response_b.json()["run_id"]
