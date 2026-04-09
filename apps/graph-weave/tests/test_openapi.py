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
