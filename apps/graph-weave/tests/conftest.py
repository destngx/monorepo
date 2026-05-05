import pytest
import json
import os
from pathlib import Path
import httpx
from src.adapters.workflow import RedisWorkflowStore
from src.modules.shared import deps
from src.adapters.cache import MockRedisAdapter

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
    # Only mock for unit tests. E2E tests run as real integrations.
    if "tests/unit" not in str(request.node.fspath):
        yield
        return

    # Ensure URL is empty to trigger MockRedisAdapter in deps.py
    monkeypatch.setattr(deps.GraphWeaveConfig, "UPSTASH_REDIS_REST_URL", "")
    monkeypatch.setattr(deps.GraphWeaveConfig, "UPSTASH_REDIS_REST_TOKEN", "")
    
    # Force a refresh of the services global if it was already initialized
    deps._services = None
    services = deps.get_services()
    
    # Ensure it's using the non-persistent mock for tests
    if not isinstance(services.cache, MockRedisAdapter):
        services.cache = MockRedisAdapter()
        
    # Clear the cache before the test
    services.cache.clear()
    
    from src.adapters.redis_circuit_breaker import NamespacedRedisClient, FallbackStorage
    services.redis_client = NamespacedRedisClient(
        redis_client=services.cache,
        fallback_storage=FallbackStorage()
    )
    
    services.checkpoint_service = deps.CheckpointService(services.redis_client)
    services.thread_lifecycle_service = deps.ThreadLifecycleService(services.redis_client)
    services.workflow_store = RedisWorkflowStore(services.redis_client)
    
    # Update global services
    deps._services = services
    
    yield
    
    # Clear again after test to avoid leakage
    if deps._services and hasattr(deps._services.cache, "clear"):
        deps._services.cache.clear()


@pytest.fixture(autouse=True)
def mock_gateway_http(monkeypatch, request):
    """Mocks all AI Gateway HTTP calls for unit tests."""
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

    def fake_post(url, json_body=None, **kwargs):
        # Default mock response following OpenAI structure
        content_dict = {"status": "completed", "message": "E2E Success"}
        
        # Simple routing for E2E tests based on prompt keywords
        messages = json_body.get("messages", []) if json_body else []
        prompt_texts = [m["content"].lower() for m in messages]
        full_text = " ".join(prompt_texts)
        
        if "stagnation" in full_text:
            content_dict["stagnation_detected"] = True
        if "detect_alert_storm" in full_text or "alert storm" in full_text:
            content_dict["alert_storm_detected"] = True
        if "redact" in full_text or "sensitive" in full_text:
            content_dict["redaction_completed"] = True
            content_dict["message"] = "Incident report with [REDACTED] values"
            
        content = json.dumps(content_dict)
            
        return FakeResponse({
            "choices": [{"message": {"role": "assistant", "content": content}}],
            "usage": {"total_tokens": 1},
            "model": json_body.get("model", "gpt-4.1") if json_body else "gpt-4.1",
        })

    class FakeClient:
        def __enter__(self):
            return self
        def __exit__(self, *args):
            pass
        def post(self, url, **kwargs):
            # Capture the json argument and rename it to json_body for fake_post
            payload = kwargs.get("json")
            return fake_post(url, json_body=payload, **kwargs)

    monkeypatch.setattr(httpx, "post", fake_post)
    monkeypatch.setattr(httpx, "Client", lambda **kwargs: FakeClient())


@pytest.fixture
def mock_mcp_router():
    """Provides a mocked MCPRouter."""
    from src.adapters.mcp_router import MCPRouter
    return MCPRouter()


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
