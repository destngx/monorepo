import pytest
import json
import os
from pathlib import Path
import httpx
from src.adapters.workflow import MockWorkflowStore
from src.modules.shared import deps
from src.adapters.cache import MockRedisAdapter

collect_ignore_glob = ["test_*.py"]


os.environ.setdefault("GITHUB_TOKEN", "gho_test_token")


def _load_dotenv_local():
    env_path = Path(__file__).resolve().parent.parent / ".env.local"
    if not env_path.exists():
        return
    with open(env_path) as f:
        for raw_line in f:
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            if key and key not in os.environ:
                os.environ[key] = value


_load_dotenv_local()

LIVE_E2E = os.getenv("GRAPH_WEAVE_LIVE_E2E", "false").lower() == "true"


class FakeRedisClient:
    def __init__(self):
        self._store = {}

    def set(self, key, value):
        self._store[key] = value

    def get(self, key):
        return self._store.get(key)

    def delete(self, key):
        self._store.pop(key, None)

    def exists(self, key):
        return key in self._store


@pytest.fixture(autouse=True)
def clear_workflow_store():
    store = MockWorkflowStore()
    store.clear()
    yield
    store.clear()


@pytest.fixture(autouse=True)
def mock_redis_services(monkeypatch, request):
    if request.node.get_closest_marker("live") is not None:
        return
    monkeypatch.setattr(deps.GraphWeaveConfig, "UPSTASH_REDIS_REST_URL", "test")
    monkeypatch.setattr(deps.GraphWeaveConfig, "UPSTASH_REDIS_REST_TOKEN", "test")
    monkeypatch.setenv("GITHUB_TOKEN", "gho_test_token")
    monkeypatch.setattr(
        deps.RedisAdapter,
        "from_env",
        classmethod(lambda cls, url, token: MockRedisAdapter()),
    )
    services = deps.Services()
    services.cache = MockRedisAdapter()
    services.checkpoint_service = deps.CheckpointService(services.cache)
    services.thread_lifecycle_service = deps.ThreadLifecycleService(services.cache)
    deps._services = services


@pytest.fixture(autouse=True)
def mock_github_provider_http(monkeypatch, request):
    if request.node.get_closest_marker("live") is not None:
        return

    class FakeResponse:
        def raise_for_status(self):
            return None

        def json(self):
            return {
                "choices": [{"message": {"content": '{"status": "completed"}'}}],
                "usage": {"total_tokens": 1},
                "model": "claude-3.5-sonnet",
            }

    def fake_post(*args, **kwargs):
        return FakeResponse()

    monkeypatch.setattr(httpx, "post", fake_post)


@pytest.fixture
def workflow_multi_node():
    fixture_path = (
        Path(__file__).parent / "fixtures" / "workflow_example_multi_node.json"
    )
    with open(fixture_path) as f:
        return json.load(f)


@pytest.fixture
def test_tenant_id():
    return "test-tenant"


@pytest.fixture
def test_workflow_id():
    return "multi-node-test:v1.0.0"


@pytest.fixture
def test_input_data():
    return {"topic": "earnings analysis"}
