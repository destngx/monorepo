import json as json_module
import os
from pathlib import Path
from types import SimpleNamespace

import httpx
import pytest

from src.adapters.cache import MockRedisAdapter
from src.adapters.checkpoint import RedisCheckpointStore
from src.adapters.workflow import RedisWorkflowStore
from src.adapters.schedule import RedisScheduleStore
from src.adapters.langgraph import RealLangGraphExecutor
from src.adapters.redis_circuit_breaker import FallbackStorage, NamespacedRedisClient
from src.modules.shared import deps

collect_ignore_glob = ["test_*.py"]


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

os.environ.setdefault(
    "FS_TOOL_TRASH_PATH",
    str(Path(__file__).resolve().parent.parent / "tmp" / "graph-weave-trash"),
)


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
def mock_redis_services(monkeypatch, request):
    """Mocks Redis services for unit tests and ensures a clean state."""
    if "tests/unit" not in str(request.node.fspath):
        yield
        return

    cache = MockRedisAdapter()
    redis_client = NamespacedRedisClient(
        redis_client=cache,
        fallback_storage=FallbackStorage(),
    )
    workflow_store = RedisWorkflowStore(redis_client)
    checkpoint_service = deps.CheckpointService(redis_client)
    checkpoint_store = RedisCheckpointStore(redis_client)
    thread_lifecycle_service = deps.ThreadLifecycleService(redis_client)
    schedule_store = RedisScheduleStore(redis_client)

    class MockMCPRouter:
        def parse_tool_calls(self, content, allowed_tools=None):
            return []

        def get_tool_definitions(self, allowed_tools=None):
            return []

        def execute_tool(self, tool_name, tool_args):
            return None

    mcp_router = MockMCPRouter()
    executor = RealLangGraphExecutor(
        mcp_router=mcp_router,
        redis_client=redis_client,
        checkpoint_service=checkpoint_service,
    )
    scheduler_service = deps.SchedulerService(
        schedule_store, execution_handler=None
    )

    services = SimpleNamespace(
        cache=cache,
        redis_client=redis_client,
        workflow_store=workflow_store,
        checkpoint_service=checkpoint_service,
        checkpoint_store=checkpoint_store,
        thread_lifecycle_service=thread_lifecycle_service,
        mcp_router=mcp_router,
        executor=executor,
        schedule_store=schedule_store,
        scheduler_service=scheduler_service,
    )
    deps._services = services

    yield

    if hasattr(cache, "clear"):
        cache.clear()


@pytest.fixture(autouse=True)
def clear_workflow_store(mock_redis_services):
    store = deps.get_workflow_store()
    store.clear()
    yield
    store.clear()


@pytest.fixture(autouse=True)
def mock_gateway_http(monkeypatch, request):
    """Mocks AI Gateway HTTP calls for unit tests only."""
    if "tests/unit" not in str(request.node.fspath):
        return

    class FakeResponse:
        def __init__(self, json_data):
            self.json_data = json_data
            self.status_code = 200

        def raise_for_status(self):
            return None

        def json(self):
            return self.json_data

    original_post = httpx.Client.post

    def fake_post(self, url, headers=None, json=None, **kwargs):
        if "chat/completions" not in str(url):
            return original_post(self, url, headers=headers, json=json, **kwargs)

        payload = json or {}
        messages = payload.get("messages", [])
        prompt_texts = [m["content"].lower() for m in messages]
        full_text = " ".join(prompt_texts)

        content_dict = {"status": "completed", "message": "E2E Success"}
        if "stagnation" in full_text:
            content_dict["stagnation_detected"] = True
        if "detect_alert_storm" in full_text or "alert storm" in full_text:
            content_dict["alert_storm_detected"] = True
        if "redact" in full_text or "sensitive" in full_text:
            content_dict["redaction_completed"] = True
            content_dict["message"] = "Incident report with [REDACTED] values"

        content = json_module.dumps(content_dict)
        return FakeResponse(
            {
                "choices": [
                    {"message": {"role": "assistant", "content": content}}
                ],
                "usage": {"total_tokens": 1},
                "model": payload.get("model", "gpt-4.1"),
            }
        )

    monkeypatch.setattr(httpx.Client, "post", fake_post)


@pytest.fixture
def mock_mcp_router():
    """Provides a mocked MCPRouter."""
    class MockMCPRouter:
        def parse_tool_calls(self, content, allowed_tools=None):
            return []

        def get_tool_definitions(self, allowed_tools=None):
            return []

        def execute_tool(self, tool_name, tool_args):
            return {"tool_name": tool_name, "arguments": tool_args}

        def filter_allowed_tools(self, tools, allowed_tools=None):
            return tools or []

    return MockMCPRouter()


@pytest.fixture
def workflow_multi_node():
    fixture_path = Path(__file__).parent / "fixtures" / "workflow_example_multi_node.json"
    with open(fixture_path) as f:
        return json_module.load(f)


@pytest.fixture
def test_tenant_id():
    return "test-tenant"


@pytest.fixture
def test_workflow_id():
    return "multi-node-test:v1.0.0"


@pytest.fixture
def test_input_data():
    return {"topic": "earnings analysis"}
