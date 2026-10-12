import pytest
from fastapi import FastAPI
from fastapi.testclient import TestClient

from src.ui import UI_MOUNT_PATH, mount_ui


@pytest.fixture
def ui_build(tmp_path):
    (tmp_path / "index.html").write_text("<html>graphweave shell</html>")
    (tmp_path / "_app").mkdir()
    (tmp_path / "_app" / "start.js").write_text("console.log('start');")
    return tmp_path


def make_client(build_dir) -> TestClient:
    app = FastAPI()

    @app.get("/workflows")
    async def workflows():
        return {"workflows": []}

    mount_ui(app, build_dir)
    return TestClient(app)


class TestUiServing:
    def test_mount_path_is_ui(self):
        assert UI_MOUNT_PATH == "/ui"

    def test_serves_index_at_mount_root(self, ui_build):
        response = make_client(ui_build).get("/ui/")
        assert response.status_code == 200
        assert "graphweave shell" in response.text

    def test_serves_static_assets(self, ui_build):
        response = make_client(ui_build).get("/ui/_app/start.js")
        assert response.status_code == 200
        assert "start" in response.text

    def test_unknown_client_route_falls_back_to_index(self, ui_build):
        response = make_client(ui_build).get("/ui/workflows/some-id")
        assert response.status_code == 200
        assert "graphweave shell" in response.text

    def test_missing_asset_returns_404_not_index(self, ui_build):
        response = make_client(ui_build).get("/ui/_app/missing.js")
        assert response.status_code == 404

    def test_root_redirects_to_ui(self, ui_build):
        response = make_client(ui_build).get("/", follow_redirects=False)
        assert response.status_code == 307
        assert response.headers["location"] == "/ui/"

    def test_api_routes_take_priority(self, ui_build):
        response = make_client(ui_build).get("/workflows")
        assert response.status_code == 200
        assert response.json() == {"workflows": []}

    def test_unknown_api_path_stays_json_404(self, ui_build):
        response = make_client(ui_build).get("/does-not-exist")
        assert response.status_code == 404
        assert response.headers["content-type"] == "application/json"


class TestUiBuildMissing:
    def test_returns_false_and_does_not_mount(self, tmp_path):
        app = FastAPI()
        assert mount_ui(app, tmp_path / "absent") is False
        client = TestClient(app)
        assert client.get("/ui/").status_code == 404
        assert client.get("/", follow_redirects=False).status_code == 404

    def test_returns_true_when_mounted(self, ui_build):
        assert mount_ui(FastAPI(), ui_build) is True
