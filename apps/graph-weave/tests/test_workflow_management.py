import pytest
from fastapi.testclient import TestClient
from src.main import app
from src.models import WorkflowCreate


@pytest.fixture
def client():
    return TestClient(app)


@pytest.fixture
def sample_workflow_definition():
    return {
        "nodes": [
            {"id": "entry", "type": "entry_node", "config": {}},
            {
                "id": "agent_1",
                "type": "agent_node",
                "config": {
                    "system_prompt": "You are a research assistant",
                    "user_prompt_template": "Research {query}",
                },
            },
            {
                "id": "exit",
                "type": "exit_node",
                "config": {"output_mapping": "$.result"},
            },
        ],
        "edges": [
            {"from": "entry", "to": "agent_1"},
            {"from": "agent_1", "to": "exit"},
        ],
        "entry_point": "entry",
        "exit_point": "exit",
    }


@pytest.fixture
def sample_create_request(sample_workflow_definition):
    return {
        "tenant_id": "hedge_fund_research_desk",
        "workflow_id": "quant-research:v3.0.0",
        "name": "Quantitative Research Pipeline",
        "version": "3.0.0",
        "description": "Comprehensive quantitative research workflow for equities analysis",
        "tags": ["research", "equities", "earnings"],
        "owner": "research_team",
        "definition": sample_workflow_definition,
    }


class TestWorkflowCreateEndpoint:
    def test_create_workflow_success(self, client, sample_create_request):
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 201
        data = response.json()
        assert data["workflow_id"] == "quant-research:v3.0.0"
        assert data["tenant_id"] == "hedge_fund_research_desk"
        assert data["name"] == "Quantitative Research Pipeline"
        assert data["status"] == "active"
        assert "created_at" in data
        assert "updated_at" in data

    def test_create_workflow_duplicate_returns_409(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 409
        data = response.json()
        assert "Conflict" in str(data)

    def test_create_workflow_invalid_workflow_id_format(
        self, client, sample_create_request
    ):
        sample_create_request["workflow_id"] = "invalid-format"
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 422

    def test_create_workflow_version_mismatch(self, client, sample_create_request):
        sample_create_request["version"] = "1.0.0"
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 422

    def test_create_workflow_missing_tenant_id(self, client, sample_create_request):
        del sample_create_request["tenant_id"]
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 422

    def test_create_workflow_missing_required_definition_fields(
        self, client, sample_create_request
    ):
        sample_create_request["definition"] = {"nodes": []}
        response = client.post("/workflows", json=sample_create_request)
        assert response.status_code == 422

    def test_create_workflow_includes_timestamps(self, client, sample_create_request):
        response = client.post("/workflows", json=sample_create_request)
        data = response.json()
        assert "Z" in data["created_at"]
        assert "Z" in data["updated_at"]
        assert data["created_at"] == data["updated_at"]

    def test_create_workflow_default_empty_tags(self, client, sample_create_request):
        sample_create_request.pop("tags")
        response = client.post("/workflows", json=sample_create_request)
        data = response.json()
        assert data["tags"] == []


class TestWorkflowGetEndpoint:
    def test_get_workflow_success(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["workflow_id"] == sample_create_request["workflow_id"]
        assert "definition" in data

    def test_get_workflow_not_found(self, client):
        response = client.get(
            "/workflows/nonexistent:v1.0.0",
            params={"tenant_id": "hedge_fund_research_desk"},
        )
        assert response.status_code == 404

    def test_get_workflow_missing_tenant_id(self, client):
        response = client.get("/workflows/test:v1.0.0")
        assert response.status_code == 400

    def test_get_workflow_tenant_scoping(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": "different_tenant"},
        )
        assert response.status_code == 404

    def test_get_workflow_includes_full_definition(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        data = response.json()
        assert "definition" in data
        assert "nodes" in data["definition"]
        assert "edges" in data["definition"]


class TestWorkflowListEndpoint:
    def test_list_workflows_empty(self, client):
        response = client.get(
            "/workflows", params={"tenant_id": "hedge_fund_research_desk"}
        )
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 0
        assert data["workflows"] == []

    def test_list_workflows_success(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            "/workflows",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 1
        assert len(data["workflows"]) == 1
        assert (
            data["workflows"][0]["workflow_id"] == sample_create_request["workflow_id"]
        )

    def test_list_workflows_no_definition_in_summary(
        self, client, sample_create_request
    ):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            "/workflows",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        data = response.json()
        assert "definition" not in data["workflows"][0]

    def test_list_workflows_filter_by_status(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "status": "active",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 1

        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "status": "archived",
            },
        )
        data = response.json()
        assert data["count"] == 0

    def test_list_workflows_filter_by_owner(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "owner": "research_team",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 1

        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "owner": "different_owner",
            },
        )
        data = response.json()
        assert data["count"] == 0

    def test_list_workflows_filter_by_tags_or_logic(
        self, client, sample_create_request
    ):
        client.post("/workflows", json=sample_create_request)
        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "tags": "research,earnings",
            },
        )
        assert response.status_code == 200
        data = response.json()
        assert data["count"] == 1

        response = client.get(
            "/workflows",
            params={
                "tenant_id": sample_create_request["tenant_id"],
                "tags": "nonexistent_tag",
            },
        )
        data = response.json()
        assert data["count"] == 0

    def test_list_workflows_missing_tenant_id(self, client):
        response = client.get("/workflows")
        assert response.status_code == 400

    def test_list_workflows_tenant_isolation(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.get("/workflows", params={"tenant_id": "different_tenant"})
        data = response.json()
        assert data["count"] == 0


class TestWorkflowUpdateEndpoint:
    def test_update_workflow_success(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        update_data = {
            "name": "Updated Research Pipeline",
            "description": "Updated description",
        }
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json=update_data,
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "Updated Research Pipeline"
        assert data["description"] == "Updated description"

    def test_update_workflow_not_found(self, client):
        response = client.put(
            "/workflows/nonexistent:v1.0.0",
            params={"tenant_id": "hedge_fund_research_desk"},
            json={"name": "New Name"},
        )
        assert response.status_code == 404

    def test_update_workflow_no_fields_provided(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json={},
        )
        assert response.status_code == 400

    def test_update_workflow_status(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json={"status": "archived"},
        )
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "archived"

    def test_update_workflow_invalid_status(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json={"status": "invalid_status"},
        )
        assert response.status_code == 422

    def test_update_workflow_updated_at_changes(self, client, sample_create_request):
        post_response = client.post("/workflows", json=sample_create_request)
        created_at = post_response.json()["created_at"]

        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json={"name": "New Name"},
        )
        data = response.json()
        assert data["created_at"] == created_at
        assert data["updated_at"] > created_at

    def test_update_workflow_missing_tenant_id(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            json={"name": "New Name"},
        )
        assert response.status_code == 400

    def test_update_workflow_multiple_fields(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        update_data = {
            "name": "New Name",
            "tags": ["new", "tags"],
            "owner": "new_owner",
            "status": "draft",
        }
        response = client.put(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
            json=update_data,
        )
        assert response.status_code == 200
        data = response.json()
        assert data["name"] == "New Name"
        assert data["tags"] == ["new", "tags"]
        assert data["owner"] == "new_owner"
        assert data["status"] == "draft"


class TestWorkflowDeleteEndpoint:
    def test_delete_workflow_success(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.delete(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        assert response.status_code == 204

        get_response = client.get(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        assert get_response.status_code == 404

    def test_delete_workflow_not_found(self, client):
        response = client.delete(
            "/workflows/nonexistent:v1.0.0",
            params={"tenant_id": "hedge_fund_research_desk"},
        )
        assert response.status_code == 404

    def test_delete_workflow_missing_tenant_id(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.delete(f"/workflows/{sample_create_request['workflow_id']}")
        assert response.status_code == 400

    def test_delete_workflow_tenant_isolation(self, client, sample_create_request):
        client.post("/workflows", json=sample_create_request)
        response = client.delete(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": "different_tenant"},
        )
        assert response.status_code == 404

        get_response = client.get(
            f"/workflows/{sample_create_request['workflow_id']}",
            params={"tenant_id": sample_create_request["tenant_id"]},
        )
        assert get_response.status_code == 200
