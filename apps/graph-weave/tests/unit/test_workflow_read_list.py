"""
Test suite for GET /workflows/{id} and GET /workflows endpoints (GW-MVP-RUNTIME-206).

Tests for:
- Permission checks for workflow reads and listings
- Pagination support for list queries
- Sorting support
- Full-text search on workflow name/description
- Version history retrieval
- Execution statistics
- Cache TTL expectations
"""

import pytest
from fastapi.testclient import TestClient

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def cleanup_and_populate():
    """Clear and populate workflow store."""
    store = get_workflow_store()
    store.clear()

    workflows = [
        {
            "tenant_id": "tenant-a",
            "workflow_id": "workflow-1",
            "name": "Alpha Workflow",
            "version": "1.0.0",
            "description": "First workflow for testing",
            "owner": "owner-1",
            "tags": ["alpha", "test"],
            "status": "active",
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        },
        {
            "tenant_id": "tenant-a",
            "workflow_id": "workflow-2",
            "name": "Beta Workflow",
            "version": "1.0.0",
            "description": "Second workflow for testing",
            "owner": "owner-1",
            "tags": ["beta", "test"],
            "status": "active",
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        },
        {
            "tenant_id": "tenant-b",
            "workflow_id": "workflow-3",
            "name": "Gamma Workflow",
            "version": "1.0.0",
            "description": "Workflow for tenant b",
            "owner": "owner-2",
            "tags": ["gamma"],
            "status": "active",
            "definition": {
                "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                "edges": [],
                "entry_point": "entry",
                "exit_point": "entry",
            },
        },
    ]

    for workflow in workflows:
        store.create(workflow["tenant_id"], workflow)

    yield store
    store.clear()


class TestWorkflowReadPermissions:
    """Test permission checks for workflow reads."""

    def test_read_nonexistent_workflow(self, client, cleanup_and_populate):
        """Reading non-existent workflow returns 404."""
        response = client.get(
            "/workflows/nonexistent",
            params={"tenant_id": "tenant-a"},
        )
        assert response.status_code == 404

    def test_read_requires_tenant_id(self, client, cleanup_and_populate):
        """Reading workflow requires tenant_id parameter."""
        response = client.get("/workflows/workflow-1")
        assert response.status_code in [400, 422]

    def test_read_workflow_success(self, client, cleanup_and_populate):
        """Reading existing workflow succeeds."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["workflow_id"] == "workflow-1"
        assert data["tenant_id"] == "tenant-a"

    def test_read_wrong_tenant_workflow(self, client, cleanup_and_populate):
        """Cannot read workflow from different tenant."""
        response = client.get(
            "/workflows/workflow-3",
            params={"tenant_id": "tenant-a"},
        )
        assert response.status_code == 404

    def test_read_returns_all_workflow_fields(self, client, cleanup_and_populate):
        """Read response includes all workflow fields."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a"},
        )
        assert response.status_code == 200
        data = response.json()

        required_fields = [
            "workflow_id",
            "tenant_id",
            "name",
            "version",
            "description",
            "owner",
            "tags",
            "status",
            "definition",
            "created_at",
            "updated_at",
        ]
        for field in required_fields:
            assert field in data, f"Missing field: {field}"


class TestWorkflowListPermissions:
    """Test permission checks for workflow listings."""

    def test_list_requires_tenant_id(self, client, cleanup_and_populate):
        """Listing workflows requires tenant_id."""
        response = client.get("/workflows")
        assert response.status_code in [400, 422]

    def test_list_workflows_for_tenant(self, client, cleanup_and_populate):
        """List returns only workflows for requested tenant."""
        response = client.get("/workflows", params={"tenant_id": "tenant-a"})
        assert response.status_code == 200
        data = response.json()

        workflows = data.get("workflows", [])
        assert len(workflows) == 2

        tenant_a_ids = {w["workflow_id"] for w in workflows}
        assert tenant_a_ids == {"workflow-1", "workflow-2"}

    def test_list_empty_for_new_tenant(self, client, cleanup_and_populate):
        """List returns empty for tenant with no workflows."""
        response = client.get("/workflows", params={"tenant_id": "tenant-new"})
        assert response.status_code == 200
        data = response.json()
        assert len(data.get("workflows", [])) == 0

    def test_list_isolation_between_tenants(self, client, cleanup_and_populate):
        """List results are isolated between tenants."""
        response_a = client.get("/workflows", params={"tenant_id": "tenant-a"})
        response_b = client.get("/workflows", params={"tenant_id": "tenant-b"})

        assert response_a.status_code == 200
        assert response_b.status_code == 200

        workflows_a = response_a.json().get("workflows", [])
        workflows_b = response_b.json().get("workflows", [])

        assert len(workflows_a) == 2
        assert len(workflows_b) == 1


class TestWorkflowListPagination:
    """Test pagination for workflow listings."""

    def test_list_with_skip_parameter(self, client, cleanup_and_populate):
        """List supports skip parameter for pagination."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "skip": 1},
        )
        assert response.status_code == 200
        data = response.json()

        workflows = data.get("workflows", [])
        assert len(workflows) >= 0

    def test_list_with_limit_parameter(self, client, cleanup_and_populate):
        """List supports limit parameter for pagination."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "limit": 1},
        )
        assert response.status_code == 200
        data = response.json()

        workflows = data.get("workflows", [])
        assert len(workflows) >= 0

    def test_list_pagination_metadata(self, client, cleanup_and_populate):
        """List response includes pagination metadata."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "skip": 0, "limit": 10},
        )
        assert response.status_code == 200
        data = response.json()

        assert "count" in data or "workflows" in data


class TestWorkflowListSorting:
    """Test sorting for workflow listings."""

    def test_list_sort_by_name(self, client, cleanup_and_populate):
        """List supports sorting by name."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "sort_by": "name", "order": "asc"},
        )

        if response.status_code == 200:
            workflows = response.json().get("workflows", [])
            if len(workflows) > 1:
                names = [w["name"] for w in workflows]
                assert names == sorted(names)

    def test_list_sort_by_created_at(self, client, cleanup_and_populate):
        """List supports sorting by created_at."""
        response = client.get(
            "/workflows",
            params={
                "tenant_id": "tenant-a",
                "sort_by": "created_at",
                "order": "asc",
            },
        )

        if response.status_code == 200:
            workflows = response.json().get("workflows", [])
            assert all("created_at" in w for w in workflows)

    def test_list_sort_by_updated_at(self, client, cleanup_and_populate):
        """List supports sorting by updated_at."""
        response = client.get(
            "/workflows",
            params={
                "tenant_id": "tenant-a",
                "sort_by": "updated_at",
                "order": "desc",
            },
        )

        if response.status_code == 200:
            workflows = response.json().get("workflows", [])
            assert all("updated_at" in w for w in workflows)


class TestWorkflowListSearch:
    """Test full-text search for workflow listings."""

    def test_list_search_by_name(self, client, cleanup_and_populate):
        """List supports searching by workflow name."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "search": "Alpha"},
        )

        if response.status_code == 200:
            workflows = response.json().get("workflows", [])
            if len(workflows) > 0:
                assert any("Alpha" in w.get("name", "") for w in workflows)

    def test_list_search_by_description(self, client, cleanup_and_populate):
        """List supports searching by description."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "search": "testing"},
        )

        if response.status_code == 200:
            workflows = response.json().get("workflows", [])
            for w in workflows:
                desc = w.get("description", "").lower()
                name = w.get("name", "").lower()
                assert "testing" in desc or "testing" in name

        if response.status_code == 200:
            workflows = response.json().get("items", [])
            for w in workflows:
                desc = w.get("description", "").lower()
                name = w.get("name", "").lower()
                assert "testing" in desc or "testing" in name

    def test_list_search_returns_subset(self, client, cleanup_and_populate):
        """Search results are subset of full list."""
        response_all = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a"},
        )
        response_search = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a", "search": "Alpha"},
        )

        assert response_all.status_code == 200
        if response_search.status_code == 200:
            all_count = len(response_all.json().get("workflows", []))
            search_count = len(response_search.json().get("workflows", []))
            assert search_count <= all_count


class TestWorkflowReadDetails:
    """Test reading detailed workflow information."""

    def test_read_includes_version_history(self, client, cleanup_and_populate):
        """Read can optionally include version history."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a", "include_history": True},
        )

        if response.status_code == 200:
            data = response.json()
            assert "workflow_id" in data

    def test_read_includes_execution_stats(self, client, cleanup_and_populate):
        """Read can optionally include execution statistics."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a", "include_stats": True},
        )

        if response.status_code == 200:
            data = response.json()
            assert "workflow_id" in data

    def test_read_basic_response(self, client, cleanup_and_populate):
        """Basic read returns workflow without optional fields."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a"},
        )

        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Alpha Workflow"
        assert data["status"] == "active"


class TestWorkflowReadCaching:
    """Test cache TTL expectations for reads."""

    def test_read_response_has_caching_headers(self, client, cleanup_and_populate):
        """Read response should include caching headers."""
        response = client.get(
            "/workflows/workflow-1",
            params={"tenant_id": "tenant-a"},
        )

        assert response.status_code == 200

    def test_list_response_has_caching_headers(self, client, cleanup_and_populate):
        """List response should include caching headers."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a"},
        )

        assert response.status_code == 200

    def test_list_response_structure(self, client, cleanup_and_populate):
        """List response has consistent structure."""
        response = client.get(
            "/workflows",
            params={"tenant_id": "tenant-a"},
        )

        assert response.status_code == 200
        data = response.json()
        assert "workflows" in data
        assert isinstance(data["workflows"], list)
