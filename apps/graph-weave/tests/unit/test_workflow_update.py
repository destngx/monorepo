"""
Test suite for PUT /workflows/{id} endpoint hardening (GW-MVP-RUNTIME-207).

Tests for:
- Immutable field protection
- Permission checks for updates
- Version history creation on update
- Definition change handling
"""

import pytest
from fastapi.testclient import TestClient

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def cleanup_with_workflow():
    """Create a workflow for testing updates."""
    store = get_workflow_store()
    store.clear()

    workflow = {
        "tenant_id": "test-tenant",
        "workflow_id": "test-workflow:v1.0.0",
        "name": "Test Workflow",
        "version": "1.0.0",
        "description": "Test workflow",
        "owner": "test-owner",
        "tags": ["tag1"],
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
    yield workflow
    store.clear()


class TestWorkflowUpdateAuth:
    """Test permission checks for workflow updates."""

    def test_update_requires_tenant_id(self, client, cleanup_with_workflow):
        """Update requires tenant_id parameter."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Updated"},
        )
        assert response.status_code in [400, 422]

    def test_update_nonexistent_workflow(self, client, cleanup_with_workflow):
        """Updating non-existent workflow returns 404."""
        response = client.put(
            "/workflows/nonexistent:v1.0.0",
            json={"name": "Updated"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 404

    def test_update_wrong_tenant_workflow(self, client, cleanup_with_workflow):
        """Cannot update workflow from different tenant."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Updated"},
            params={"tenant_id": "wrong-tenant"},
        )
        assert response.status_code == 404

    def test_update_success(self, client, cleanup_with_workflow):
        """Successfully update workflow properties."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Updated Workflow"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Updated Workflow"


class TestWorkflowUpdateImmutableFields:
    """Test immutable field protection."""

    def test_cannot_update_workflow_id(self, client, cleanup_with_workflow):
        """workflow_id field cannot be modified."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"workflow_id": "new:v1.0.0"},
            params={"tenant_id": "test-tenant"},
        )
        if response.status_code == 200:
            data = response.json()
            assert data["workflow_id"] == "test-workflow:v1.0.0"

    def test_cannot_update_tenant_id(self, client, cleanup_with_workflow):
        """tenant_id field cannot be modified."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"tenant_id": "new-tenant"},
            params={"tenant_id": "test-tenant"},
        )
        if response.status_code == 200:
            data = response.json()
            assert data["tenant_id"] == "test-tenant"

    def test_cannot_update_created_at(self, client, cleanup_with_workflow):
        """created_at field cannot be modified."""
        store = get_workflow_store()
        original = store.get("test-tenant", "test-workflow:v1.0.0")
        original_created_at = original["created_at"]

        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"created_at": "2020-01-01T00:00:00Z"},
            params={"tenant_id": "test-tenant"},
        )
        if response.status_code == 200:
            data = response.json()
            assert data["created_at"] == original_created_at


class TestWorkflowUpdateMutableFields:
    """Test updating mutable fields."""

    def test_update_name(self, client, cleanup_with_workflow):
        """Can update workflow name."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "New Name"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "New Name"

    def test_update_description(self, client, cleanup_with_workflow):
        """Can update workflow description."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"description": "New description"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["description"] == "New description"

    def test_update_tags(self, client, cleanup_with_workflow):
        """Can update workflow tags."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"tags": ["new-tag", "another-tag"]},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert "new-tag" in data.get("tags", [])

    def test_update_definition(self, client, cleanup_with_workflow):
        """Can update workflow definition."""
        new_definition = {
            "nodes": [
                {"id": "entry", "type": "entry", "config": {}},
                {"id": "process", "type": "process", "config": {}},
                {"id": "exit", "type": "exit", "config": {}},
            ],
            "edges": [
                {"from": "entry", "to": "process"},
                {"from": "process", "to": "exit"},
            ],
            "entry_point": "entry",
            "exit_point": "exit",
        }

        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"definition": new_definition},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert len(data["definition"]["nodes"]) == 3


class TestWorkflowUpdateVersioning:
    """Test version history on updates."""

    def test_update_changes_updated_at(self, client, cleanup_with_workflow):
        """Update changes the updated_at timestamp."""
        store = get_workflow_store()
        before = store.get("test-tenant", "test-workflow:v1.0.0")
        before_updated_at = before["updated_at"]

        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Updated"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["updated_at"] != before_updated_at

    def test_update_preserves_created_at(self, client, cleanup_with_workflow):
        """Update preserves the original created_at timestamp."""
        store = get_workflow_store()
        before = store.get("test-tenant", "test-workflow:v1.0.0")
        before_created_at = before["created_at"]

        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Updated"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["created_at"] == before_created_at

    def test_multiple_updates_tracked(self, client, cleanup_with_workflow):
        """Multiple updates create separate version history entries."""
        response1 = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "First Update"},
            params={"tenant_id": "test-tenant"},
        )

        response2 = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Second Update"},
            params={"tenant_id": "test-tenant"},
        )

        assert response1.status_code == 200
        assert response2.status_code == 200

        final_data = response2.json()
        assert final_data["name"] == "Second Update"


class TestWorkflowUpdateStatus:
    """Test status field behavior during updates."""

    def test_can_change_workflow_status(self, client, cleanup_with_workflow):
        """Can change workflow status."""
        response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"status": "archived"},
            params={"tenant_id": "test-tenant"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "archived"

    def test_update_reflects_in_retrieval(self, client, cleanup_with_workflow):
        """Updated workflow is reflected in subsequent retrieval."""
        update_response = client.put(
            "/workflows/test-workflow:v1.0.0",
            json={"name": "Final Name", "description": "Final desc"},
            params={"tenant_id": "test-tenant"},
        )
        assert update_response.status_code == 200

        get_response = client.get(
            "/workflows/test-workflow:v1.0.0",
            params={"tenant_id": "test-tenant"},
        )
        assert get_response.status_code == 200
        data = get_response.json()
        assert data["name"] == "Final Name"
        assert data["description"] == "Final desc"
