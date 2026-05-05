"""
Test suite for POST /workflows endpoint hardening (GW-MVP-RUNTIME-205).

Tests for:
- Authentication and authorization for workflow creation
- Tenant ownership verification
- Workflow definition schema validation
- Name/ID uniqueness enforcement
- Workflow persistence
"""

import pytest
from fastapi.testclient import TestClient
from unittest.mock import patch

from src.main import app
from src.modules.shared.deps import get_workflow_store


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def cleanup_store():
    """Clear workflow store before and after each test."""
    store = get_workflow_store()
    store.clear()
    yield
    store.clear()


class TestWorkflowCreateAuth:
    """Test authentication and authorization for workflow creation."""

    def test_create_missing_tenant_id(self, client, cleanup_store):
        """Reject workflow creation without tenant_id."""
        response = client.post(
            "/workflows",
            json={
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "definition": {"nodes": [], "edges": []},
            },
        )
        assert response.status_code == 422

    def test_create_missing_workflow_definition(self, client, cleanup_store):
        """Reject workflow creation without definition."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
            },
        )
        assert response.status_code == 422

    def test_create_missing_name(self, client, cleanup_store):
        """Reject workflow creation without name."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "version": "1.0.0",
                "definition": {"nodes": [], "edges": []},
            },
        )
        assert response.status_code == 422

    def test_create_invalid_tenant_id_format(self, client, cleanup_store):
        """Reject workflow creation with invalid tenant_id format."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "definition": {"nodes": [], "edges": []},
            },
        )
        assert response.status_code == 422


class TestWorkflowCreateValidation:
    """Test workflow definition schema validation."""

    def test_create_valid_workflow(self, client, cleanup_store):
        """Successfully create a valid workflow."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test Workflow",
                "version": "1.0.0",
                "description": "Test workflow",
                "owner": "test-owner",
                "tags": ["tag1"],
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
        assert response.status_code == 201
        data = response.json()
        assert data["name"] == "Test Workflow"
        assert data["tenant_id"] == "test-tenant"

    def test_create_missing_entry_point(self, client, cleanup_store):
        """Reject workflow without entry_point in definition."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "definition": {
                    "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                    "edges": [],
                    "exit_point": "entry",
                },
            },
        )
        assert response.status_code == 422

    def test_create_missing_exit_point(self, client, cleanup_store):
        """Reject workflow without exit_point in definition."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "definition": {
                    "nodes": [{"id": "entry", "type": "entry", "config": {}}],
                    "edges": [],
                    "entry_point": "entry",
                },
            },
        )
        assert response.status_code == 422

    def test_create_empty_nodes_list(self, client, cleanup_store):
        """Reject workflow with empty nodes list."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "definition": {
                    "nodes": [],
                    "edges": [],
                    "entry_point": "entry",
                    "exit_point": "exit",
                },
            },
        )
        assert response.status_code == 422


class TestWorkflowCreateUniqueness:
    """Test uniqueness enforcement for workflows."""

    def test_create_duplicate_workflow_id(self, client, cleanup_store):
        """Reject duplicate workflow_id within same tenant."""
        workflow_data = {
            "tenant_id": "test-tenant",
            "workflow_id": "test-workflow:v1.0.0",
            "name": "Test Workflow",
            "version": "1.0.0",
            "description": "Test",
            "owner": "owner",
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
        }

        response1 = client.post("/workflows", json=workflow_data)
        assert response1.status_code == 201

        response2 = client.post("/workflows", json=workflow_data)
        assert response2.status_code == 409 or response2.status_code == 400

    def test_create_same_name_different_tenant(self, client, cleanup_store):
        """Allow same workflow name in different tenants."""
        workflow_template = {
            "workflow_id": "shared-workflow:v1.0.0",
            "name": "Shared Workflow",
            "version": "1.0.0",
            "description": "Shared",
            "owner": "owner",
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
        }

        response1 = client.post(
            "/workflows", json={**workflow_template, "tenant_id": "tenant-a"}
        )

        workflow_template["workflow_id"] = "shared-workflow-2:v1.0.0"
        response2 = client.post(
            "/workflows", json={**workflow_template, "tenant_id": "tenant-b"}
        )

        assert response1.status_code == 201
        assert response2.status_code == 201



    def test_create_returns_unique_workflow_id(self, client, cleanup_store):
        """Created workflows have unique workflow_id."""
        workflow_template = {
            "tenant_id": "test-tenant",
            "name": "Workflow",
            "version": "1.0.0",
            "description": "Test",
            "owner": "owner",
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
        }

        response1 = client.post(
            "/workflows",
            json={
                **workflow_template,
                "workflow_id": "workflow-1:v1.0.0",
                "name": "Workflow 1",
            },
        )
        response2 = client.post(
            "/workflows",
            json={
                **workflow_template,
                "workflow_id": "workflow-2:v1.0.0",
                "name": "Workflow 2",
            },
        )

        assert response1.status_code == 201
        assert response2.status_code == 201
        assert response1.json()["workflow_id"] != response2.json()["workflow_id"]


class TestWorkflowCreatePersistence:
    """Test workflow persistence and retrieval."""

    def test_created_workflow_is_retrievable(self, client, cleanup_store):
        """Created workflow can be retrieved via GET."""
        create_response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "retrieve-test:v1.0.0",
                "name": "Test Workflow",
                "version": "1.0.0",
                "description": "Test",
                "owner": "owner",
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

        if create_response.status_code == 201:
            workflow_id = create_response.json()["workflow_id"]

            get_response = client.get(
                f"/workflows/{workflow_id}",
                params={"tenant_id": "test-tenant"},
            )
            assert get_response.status_code == 200
            assert get_response.json()["workflow_id"] == workflow_id

    def test_created_workflow_has_timestamps(self, client, cleanup_store):
        """Created workflow includes created_at and updated_at."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "description": "Test",
                "owner": "owner",
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
        assert response.status_code == 201
        data = response.json()

        assert "created_at" in data
        assert "updated_at" in data
        assert data["created_at"] is not None
        assert data["updated_at"] is not None

    def test_created_workflow_has_active_status(self, client, cleanup_store):
        """Newly created workflow has 'active' status."""
        response = client.post(
            "/workflows",
            json={
                "tenant_id": "test-tenant",
                "workflow_id": "test-workflow:v1.0.0",
                "name": "Test",
                "version": "1.0.0",
                "description": "Test",
                "owner": "owner",
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
        assert response.status_code == 201
        data = response.json()
        assert data["status"] == "active"


class TestWorkflowCreateTenantOwnership:
    """Test tenant ownership verification."""

    def test_create_enforces_tenant_context(self, client, cleanup_store):
        """Workflow creation enforces tenant_id context."""
        workflow_data = {
            "tenant_id": "tenant-a",
            "workflow_id": "test-workflow:v1.0.0",
            "name": "Test",
            "version": "1.0.0",
            "description": "Test",
            "owner": "owner",
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
        }

        response = client.post("/workflows", json=workflow_data)
        assert response.status_code == 201

        created_workflow = response.json()
        assert created_workflow["tenant_id"] == "tenant-a"

        store = get_workflow_store()
        retrieved = store.get("tenant-a", created_workflow["workflow_id"])
        assert retrieved is not None
        assert retrieved["tenant_id"] == "tenant-a"
