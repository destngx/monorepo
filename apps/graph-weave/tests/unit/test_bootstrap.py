import pytest
from fastapi.testclient import TestClient
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestAppBootstrap:
    def test_app_shell_boots(self, client, monkeypatch):
        # Mock httpx.get for the gateway health check
        import httpx

        class MockResponse:
            def __init__(self, status_code):
                self.status_code = status_code

        def mock_get(*args, **kwargs):
            return MockResponse(200)

        monkeypatch.setattr(httpx, "get", mock_get)

        response = client.get("/health")
        assert response.status_code == 200
        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "ok"
        assert "ai_gateway" in data["services"]
        assert "redis" in data["services"]

    def test_openapi_docs_available(self, client):
        response = client.get("/api/docs")
        assert response.status_code == 200
        assert response.json() is not None

    def test_app_has_title(self):
        assert app.title == "GraphWeave"

    def test_app_has_version(self):
        assert app.version == "0.1.0"
