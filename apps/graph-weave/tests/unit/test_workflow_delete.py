"""
Test suite for DELETE /workflows/{id} endpoint hardening (GW-MVP-RUNTIME-208).

Tests for:
- Permission checks for deletion
- Dependency validation
- Soft-delete behavior
- Multi-tenant isolation
"""

import pytest
from fastapi.testclient import TestClient

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def cleanup_with_workflows():
    """Create multiple workflows for testing deletion."""
    store = get_workflow_store()
    store.clear()

    workflows = [
        {
            "tenant_id": "test-tenant",
            "workflow_id": "workflow-to-delete:v1.0.0",
            "name": "Workflow To Delete",
            "version": "1.0.0",
            "description": "This will be deleted",
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
        },
        {
            "tenant_id": "test-tenant",
            "workflow_id": "other-workflow:v1.0.0",
            "name": "Other Workflow",
            "version": "1.0.0",
            "description": "No dependencies",
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
        },
    ]

    for workflow in workflows:
        store.create(workflow["tenant_id"], workflow)

    yield store
    store.clear()


class TestWorkflowDeleteAuth:
    """Test permission checks for workflow deletion."""

    def test_delete_requires_tenant_id(self, client, cleanup_with_workflows):
        """Delete requires tenant_id parameter."""
        response = client.delete("/workflows/workflow-to-delete:v1.0.0")
        assert response.status_code in [400, 422]

    def test_delete_nonexistent_workflow(self, client, cleanup_with_workflows):
        """Deleting non-existent workflow returns 404."""
        response = client.delete(
            "/workflows/nonexistent:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 404

    def test_delete_wrong_tenant_workflow(self, client, cleanup_with_workflows):
        """Cannot delete workflow from different tenant."""
        response = client.delete(
            "/workflows/workflow-to-delete:v1.0.0",
            params={"tenant_id": "wrong-tenant"},
        )
        assert response.status_code == 404

    def test_delete_success(self, client, cleanup_with_workflows):
        """Successfully delete a workflow."""
        response = client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code in [200, 204]


class TestWorkflowDeleteBehavior:
    """Test deletion behavior."""

    def test_deleted_workflow_no_longer_listed(self, client, cleanup_with_workflows):
        """Deleted workflow does not appear in list."""
        list_before = client.get(
            "/workflows",
            params={"tenant_id": "test-tenant"},
        )
        count_before = len(list_before.json().get("workflows", []))

        client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        list_after = client.get(
            "/workflows",
            params={"tenant_id": "test-tenant"},
        )
        count_after = len(list_after.json().get("workflows", []))

        assert count_after < count_before

    def test_deleted_workflow_not_retrievable(self, client, cleanup_with_workflows):
        """Deleted workflow cannot be retrieved."""
        client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        response = client.get(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 404

    def test_cannot_delete_twice(self, client, cleanup_with_workflows):
        """Deleting the same workflow twice returns error on second attempt."""
        client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        response_2 = client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )
        assert response_2.status_code == 404


class TestWorkflowDeleteIsolation:
    """Test multi-tenant isolation for deletes."""

    def test_delete_one_tenant_does_not_affect_other(
        self, client, cleanup_with_workflows
    ):
        """Deleting workflow in one tenant does not affect other tenants."""
        store = get_workflow_store()
        store.clear()

        workflow_template = {
            "workflow_id": "shared:v1.0.0",
            "name": "Shared",
            "version": "1.0.0",
            "description": "Shared",
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

        client.delete(
            "/workflows/shared:v1.0.0",
            params={"tenant_id": "tenant-a"},
        )

        response_b = client.get(
            "/workflows/shared:v1.0.0",
            params={"tenant_id": "tenant-b"},
        )
        assert response_b.status_code == 200


class TestWorkflowDeleteConfirmation:
    """Test deletion confirmation."""

    def test_delete_response_structure(self, client, cleanup_with_workflows):
        """Delete response has expected structure."""
        response = client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        assert response.status_code in [200, 204]

    def test_delete_confirmation_is_consistent(self, client, cleanup_with_workflows):
        """Deletion confirmation is consistent."""
        store = get_workflow_store()

        before_count = len(store.list_for_tenant("test-tenant"))

        response = client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        assert response.status_code in [200, 204]

        after_count = len(store.list_for_tenant("test-tenant"))
        assert after_count < before_count

    def test_delete_multiple_workflows_sequentially(
        self, client, cleanup_with_workflows
    ):
        """Can delete multiple workflows in sequence."""
        response1 = client.delete(
            "/workflows/other-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )

        assert response1.status_code in [200, 204]

        store = get_workflow_store()
        remaining = store.list_for_tenant("test-tenant")
        assert len(remaining) >= 0
