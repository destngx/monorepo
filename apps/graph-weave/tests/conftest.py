import pytest
import json
from pathlib import Path
from src.modules.shared.deps import get_workflow_store


@pytest.fixture(autouse=True)
def clear_workflow_store():
    store = get_workflow_store()
    store.clear()
    yield
    store.clear()


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
