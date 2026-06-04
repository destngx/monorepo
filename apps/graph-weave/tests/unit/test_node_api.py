import pytest
from unittest.mock import AsyncMock
from fastapi.testclient import TestClient
from src.main import app
from src.models.node import NodeCreate, NodeConfig, InputContract, OutputContract, ContractField
from src.adapters.node import RedisNodeStore, ConflictError, NotFoundError
from src.services.node_validator import NodeValidator
from src.modules.shared.deps import get_tenant_node_store, get_node_validator
from src.models.node import NodeListResponse


@pytest.fixture
def client():
    app.dependency_overrides.clear()
    with TestClient(app) as test_client:
        yield test_client
    app.dependency_overrides.clear()


def override_node_deps(mock_node_store):
    app.dependency_overrides[get_tenant_node_store] = lambda: mock_node_store
    app.dependency_overrides[get_node_validator] = lambda: NodeValidator()


@pytest.fixture
def mock_node_store():
    store = AsyncMock(spec=RedisNodeStore)
    return store


@pytest.fixture
def sample_node():
    return {
        "tenant_id": "test-tenant",
        "node_id": "test_node:v1.0.0",
        "node_name": "test_node",
        "version": "1.0.0",
        "name": "Test Node",
        "type": "agent_node",
        "description": "A test node",
        "config": {"system_prompt": "Test prompt"},
        "input_contract": {
            "required": [{"name": "input1", "type": "string"}],
            "optional": [],
        },
        "output_contract": {
            "produced": [
                {"name": "output1", "type": "string", "state_path": "$.test.output1"}
            ]
        },
        "tags": ["test"],
    }


def test_create_node_200(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    mock_node_store.create.return_value = NodeResponse(
        **sample_node,
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance={},
        created_at="",
        immutable_fields=["config", "input_contract", "output_contract", "type"],
    )

    override_node_deps(mock_node_store)
    response = client.post("/nodes/", json=sample_node)
    assert response.status_code == 201


def test_create_node_validation_error_422(client, mock_node_store, sample_node):
    sample_node["type"] = "invalid_type"

    override_node_deps(mock_node_store)
    response = client.post("/nodes/", json=sample_node)
    assert response.status_code == 422


def test_create_node_duplicate_409(client, mock_node_store, sample_node):
    mock_node_store.create.side_effect = ConflictError("Node already exists")
    mock_node_store.get.return_value = None

    override_node_deps(mock_node_store)
    response = client.post("/nodes/", json=sample_node)
    assert response.status_code == 409


def test_get_node_200(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    mock_node_store.get.return_value = NodeResponse(
        **sample_node,
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance={},
        created_at="",
        immutable_fields=["config", "input_contract", "output_contract", "type"],
    )

    override_node_deps(mock_node_store)
    response = client.get("/nodes/test_node:v1.0.0?tenant_id=test-tenant")
    assert response.status_code == 200


def test_get_node_404(client, mock_node_store):
    mock_node_store.get.return_value = None

    override_node_deps(mock_node_store)
    response = client.get("/nodes/nonexistent:v1.0.0?tenant_id=test-tenant")
    assert response.status_code == 404


def test_list_nodes_200(client, mock_node_store):
    mock_node_store.list.return_value = NodeListResponse(
        nodes=[],
        total=0,
        page=1,
        page_size=20,
    )

    override_node_deps(mock_node_store)
    response = client.get("/nodes/?tenant_id=test-tenant")
    assert response.status_code == 200


def test_update_node_200(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    updated_node = {**sample_node, "name": "Updated Name"}
    mock_node_store.update.return_value = NodeResponse(
        **updated_node,
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance={},
        created_at="",
        immutable_fields=["config", "input_contract", "output_contract", "type"],
    )

    override_node_deps(mock_node_store)
    response = client.put(
        "/nodes/test_node:v1.0.0?tenant_id=test-tenant",
        json={"name": "Updated Name"},
    )
    assert response.status_code == 200


def test_update_node_404(client, mock_node_store):
    mock_node_store.update.side_effect = NotFoundError("Node not found")

    override_node_deps(mock_node_store)
    response = client.put(
        "/nodes/nonexistent:v1.0.0?tenant_id=test-tenant",
        json={"name": "Updated Name"},
    )
    assert response.status_code == 404


def test_delete_node_200(client, mock_node_store):
    mock_node_store.delete.return_value = True

    override_node_deps(mock_node_store)
    response = client.delete("/nodes/test_node:v1.0.0?tenant_id=test-tenant")
    assert response.status_code == 200


def test_delete_node_404(client, mock_node_store):
    mock_node_store.delete.return_value = False

    override_node_deps(mock_node_store)
    response = client.delete("/nodes/nonexistent:v1.0.0?tenant_id=test-tenant")
    assert response.status_code == 404


def test_verify_node_intent_mismatch(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    sample_node["provenance"] = {"source_intent": "old intent context"}
    mock_node_store.get.return_value = NodeResponse(
        **sample_node,
        owner="system",
        status="active",
        reuse_eligible=True,
        created_at="",
        immutable_fields=[],
    )

    override_node_deps(mock_node_store)
    response = client.get("/nodes/verify?node_id=test_node:v1.0.0&tenant_id=test-tenant&type=agent_node&intent_context=new%20intent%20context")
    assert response.status_code == 200
    assert response.json()["is_reusable"] is False
    assert "Intent context mismatch" in response.json()["reason"]


def test_verify_node_intent_match(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    sample_node["provenance"] = {"source_intent": "same intent context"}
    mock_node_store.get.return_value = NodeResponse(
        **sample_node,
        owner="system",
        status="active",
        reuse_eligible=True,
        created_at="",
        immutable_fields=[],
    )

    override_node_deps(mock_node_store)
    response = client.get("/nodes/verify?node_id=test_node:v1.0.0&tenant_id=test-tenant&type=agent_node&intent_context=same%20intent%20context")
    assert response.status_code == 200
    assert response.json()["is_reusable"] is True

