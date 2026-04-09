import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestOpenAPIDocs:
    def test_openapi_schema_endpoint_exists(self, client):
        response = client.get("/openapi.json")
        assert response.status_code == 200

    def test_openapi_schema_is_json(self, client):
        response = client.get("/openapi.json")
        assert response.headers.get("content-type") == "application/json"

    def test_openapi_schema_has_paths(self, client):
        response = client.get("/openapi.json")
        schema = response.json()
        assert "paths" in schema
        assert isinstance(schema["paths"], dict)

    def test_openapi_schema_has_title(self, client):
        response = client.get("/openapi.json")
        schema = response.json()
        assert "info" in schema
        assert schema["info"]["title"] == "GraphWeave"

    def test_openapi_execute_endpoint_documented(self, client):
        response = client.get("/openapi.json")
        schema = response.json()
        assert "/execute" in schema["paths"]
        assert "post" in schema["paths"]["/execute"]

    def test_openapi_execute_endpoint_has_requestbody(self, client):
        response = client.get("/openapi.json")
        schema = response.json()
        execute_post = schema["paths"]["/execute"]["post"]
        assert "requestBody" in execute_post

    def test_openapi_execute_endpoint_has_response_schema(self, client):
        response = client.get("/openapi.json")
        schema = response.json()
        execute_post = schema["paths"]["/execute"]["post"]
        assert "responses" in execute_post
        assert "200" in execute_post["responses"]

    def test_swagger_ui_available(self, client):
        response = client.get("/docs")
        assert response.status_code == 200


class TestOpenAPITags:
    """Verify OpenAPI schema includes semantic tag organization"""

    def test_openapi_has_tags_array(self, client):
        """Verify OpenAPI schema includes tags array with 3 groups"""
        response = client.get("/openapi.json")
        schema = response.json()
        assert "tags" in schema
        assert len(schema["tags"]) == 3

    def test_openapi_tag_definitions(self, client):
        """Verify each tag has name and description"""
        response = client.get("/openapi.json")
        schema = response.json()
        tag_names = {tag["name"] for tag in schema["tags"]}

        assert tag_names == {"Execution", "Skills", "Workflows"}

        for tag in schema["tags"]:
            assert "name" in tag
            assert "description" in tag
            assert len(tag["description"]) > 0

    def test_workflows_tag_has_correct_description(self, client):
        """Verify Workflows tag has correct description"""
        response = client.get("/openapi.json")
        schema = response.json()
        workflows_tag = next(t for t in schema["tags"] if t["name"] == "Workflows")
        assert (
            "CRUD" in workflows_tag["description"]
            or "workflow" in workflows_tag["description"].lower()
        )

    def test_execution_tag_has_correct_description(self, client):
        """Verify Execution tag has correct description"""
        response = client.get("/openapi.json")
        schema = response.json()
        execution_tag = next(t for t in schema["tags"] if t["name"] == "Execution")
        assert "execution" in execution_tag["description"].lower()

    def test_skills_tag_has_correct_description(self, client):
        """Verify Skills tag has correct description"""
        response = client.get("/openapi.json")
        schema = response.json()
        skills_tag = next(t for t in schema["tags"] if t["name"] == "Skills")
        assert "skill" in skills_tag["description"].lower()


class TestWorkflowEndpointTags:
    """Verify workflow endpoints have Workflows tag"""

    def test_post_workflows_has_workflows_tag(self, client):
        """Verify POST /workflows is tagged with Workflows"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/workflows"]["post"]
        assert "tags" in endpoint
        assert "Workflows" in endpoint["tags"]

    def test_get_workflows_list_has_workflows_tag(self, client):
        """Verify GET /workflows is tagged with Workflows"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/workflows"]["get"]
        assert "tags" in endpoint
        assert "Workflows" in endpoint["tags"]

    def test_get_workflow_detail_has_workflows_tag(self, client):
        """Verify GET /workflows/{workflow_id} is tagged with Workflows"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/workflows/{workflow_id}"]["get"]
        assert "tags" in endpoint
        assert "Workflows" in endpoint["tags"]

    def test_put_workflow_has_workflows_tag(self, client):
        """Verify PUT /workflows/{workflow_id} is tagged with Workflows"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/workflows/{workflow_id}"]["put"]
        assert "tags" in endpoint
        assert "Workflows" in endpoint["tags"]

    def test_delete_workflow_has_workflows_tag(self, client):
        """Verify DELETE /workflows/{workflow_id} is tagged with Workflows"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/workflows/{workflow_id}"]["delete"]
        assert "tags" in endpoint
        assert "Workflows" in endpoint["tags"]


class TestExecutionEndpointTags:
    """Verify execution endpoints have Execution tag"""

    def test_post_execute_has_execution_tag(self, client):
        """Verify POST /execute is tagged with Execution"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/execute"]["post"]
        assert "tags" in endpoint
        assert "Execution" in endpoint["tags"]

    def test_get_execute_status_has_execution_tag(self, client):
        """Verify GET /execute/{run_id}/status is tagged with Execution"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/execute/{run_id}/status"]["get"]
        assert "tags" in endpoint
        assert "Execution" in endpoint["tags"]


class TestSkillEndpointTags:
    """Verify skill endpoints have Skills tag"""

    def test_post_invalidate_has_skills_tag(self, client):
        """Verify POST /invalidate is tagged with Skills"""
        response = client.get("/openapi.json")
        schema = response.json()
        endpoint = schema["paths"]["/invalidate"]["post"]
        assert "tags" in endpoint
        assert "Skills" in endpoint["tags"]


class TestEndpointTagCoverage:
    """Verify all endpoints are tagged and no cross-contamination"""

    def test_all_8_endpoints_have_tags(self, client):
        """Verify all 8 endpoints include tags array"""
        response = client.get("/openapi.json")
        schema = response.json()

        endpoints_to_check = [
            ("/execute", "post"),
            ("/execute/{run_id}/status", "get"),
            ("/invalidate", "post"),
            ("/workflows", "post"),
            ("/workflows", "get"),
            ("/workflows/{workflow_id}", "get"),
            ("/workflows/{workflow_id}", "put"),
            ("/workflows/{workflow_id}", "delete"),
        ]

        for path, method in endpoints_to_check:
            assert path in schema["paths"], f"Path {path} not found"
            assert method in schema["paths"][path], (
                f"Method {method} not found for {path}"
            )
            operation = schema["paths"][path][method]
            assert "tags" in operation, f"No tags for {method.upper()} {path}"
            assert len(operation["tags"]) > 0, (
                f"Empty tags array for {method.upper()} {path}"
            )

    def test_no_default_tags(self, client):
        """Verify no endpoints use 'default' tag"""
        response = client.get("/openapi.json")
        schema = response.json()

        for path, methods in schema["paths"].items():
            for method, operation in methods.items():
                if isinstance(operation, dict) and "tags" in operation:
                    assert "default" not in operation["tags"], (
                        f"Endpoint {method.upper()} {path} uses 'default' tag instead of semantic tag"
                    )

    def test_workflows_endpoints_only_have_workflows_tag(self, client):
        """Verify workflow endpoints don't have Execution or Skills tags"""
        response = client.get("/openapi.json")
        schema = response.json()

        workflow_paths = [
            ("/workflows", "post"),
            ("/workflows", "get"),
            ("/workflows/{workflow_id}", "get"),
            ("/workflows/{workflow_id}", "put"),
            ("/workflows/{workflow_id}", "delete"),
        ]

        for path, method in workflow_paths:
            operation = schema["paths"][path][method]
            tags = operation["tags"]
            assert "Workflows" in tags, f"{method.upper()} {path} missing Workflows tag"
            assert "Execution" not in tags, (
                f"{method.upper()} {path} incorrectly has Execution tag"
            )
            assert "Skills" not in tags, (
                f"{method.upper()} {path} incorrectly has Skills tag"
            )

    def test_execution_endpoints_only_have_execution_tag(self, client):
        """Verify execution endpoints don't have Workflows or Skills tags"""
        response = client.get("/openapi.json")
        schema = response.json()

        execution_paths = [
            ("/execute", "post"),
            ("/execute/{run_id}/status", "get"),
        ]

        for path, method in execution_paths:
            operation = schema["paths"][path][method]
            tags = operation["tags"]
            assert "Execution" in tags, f"{method.upper()} {path} missing Execution tag"
            assert "Workflows" not in tags, (
                f"{method.upper()} {path} incorrectly has Workflows tag"
            )
            assert "Skills" not in tags, (
                f"{method.upper()} {path} incorrectly has Skills tag"
            )

    def test_skills_endpoints_only_have_skills_tag(self, client):
        """Verify skill endpoints don't have Execution or Workflows tags"""
        response = client.get("/openapi.json")
        schema = response.json()

        operation = schema["paths"]["/invalidate"]["post"]
        tags = operation["tags"]
        assert "Skills" in tags, f"POST /invalidate missing Skills tag"
        assert "Execution" not in tags, (
            f"POST /invalidate incorrectly has Execution tag"
        )
        assert "Workflows" not in tags, (
            f"POST /invalidate incorrectly has Workflows tag"
        )
