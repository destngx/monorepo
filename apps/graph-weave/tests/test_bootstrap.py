import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestAppBootstrap:
    def test_app_shell_boots(self, client):
        response = client.get("/health")
        assert response.status_code == 200
        assert response.json() == {"status": "ok"}

    def test_openapi_docs_available(self, client):
        response = client.get("/api/docs")
        assert response.status_code == 200
        assert response.json() is not None

    def test_app_has_title(self):
        assert app.title == "GraphWeave"

    def test_app_has_version(self):
        assert app.version == "0.1.0"
