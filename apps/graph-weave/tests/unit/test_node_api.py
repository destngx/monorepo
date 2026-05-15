import pytest
from unittest.mock import AsyncMock, patch
from fastapi.testclient import TestClient
from src.main import app
from src.models.node import NodeCreate, NodeConfig, InputContract, OutputContract, ContractField
from src.adapters.node import RedisNodeStore, ConflictError, NotFoundError
from src.services.node_validator import NodeValidator


@pytest.fixture
def client():
    return TestClient(app)


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


@pytest.mark.asyncio
async def test_create_node_200(client, mock_node_store, sample_node):
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

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        with patch("src.routers.nodes.get_node_validator", return_value=NodeValidator()):
            response = client.post("/nodes/", json=sample_node)
            assert response.status_code == 201


@pytest.mark.asyncio
async def test_create_node_validation_error_400(client, mock_node_store, sample_node):
    sample_node["type"] = "invalid_type"

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        with patch("src.routers.nodes.get_node_validator", return_value=NodeValidator()):
            response = client.post("/nodes/", json=sample_node)
            assert response.status_code == 400


@pytest.mark.asyncio
async def test_create_node_duplicate_409(client, mock_node_store, sample_node):
    mock_node_store.create.side_effect = ConflictError("Node already exists")

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        with patch("src.routers.nodes.get_node_validator", return_value=NodeValidator()):
            response = client.post("/nodes/", json=sample_node)
            assert response.status_code == 409


@pytest.mark.asyncio
async def test_get_node_200(client, mock_node_store, sample_node):
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

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.get("/nodes/test_node:v1.0.0?tenant_id=test-tenant")
        assert response.status_code == 200


@pytest.mark.asyncio
async def test_get_node_404(client, mock_node_store):
    mock_node_store.get.return_value = None

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.get("/nodes/nonexistent:v1.0.0?tenant_id=test-tenant")
        assert response.status_code == 404


@pytest.mark.asyncio
async def test_list_nodes_200(client, mock_node_store):
    mock_node_store.list.return_value = nodes=total=0

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.get("/nodes/?tenant_id=test-tenant")
        assert response.status_code == 200


@pytest.mark.asyncio
async def test_update_node_200(client, mock_node_store, sample_node):
    from src.models.node import NodeResponse

    mock_node_store.update.return_value = NodeResponse(
        **sample_node,
        name="Updated Name",
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance={},
        created_at="",
        immutable_fields=["config", "input_contract", "output_contract", "type"],
    )

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.put(
            "/nodes/test_node:v1.0.0?tenant_id=test-tenant",
            json={"name": "Updated Name"},
        )
        assert response.status_code == 200


@pytest.mark.asyncio
async def test_update_node_404(client, mock_node_store):
    mock_node_store.update.side_effect = NotFoundError("Node not found")

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.put(
            "/nodes/nonexistent:v1.0.0?tenant_id=test-tenant",
            json={"name": "Updated Name"},
        )
        assert response.status_code == 404


@pytest.mark.asyncio
async def test_delete_node_200(client, mock_node_store):
    mock_node_store.delete.return_value = True

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.delete("/nodes/test_node:v1.0.0?tenant_id=test-tenant")
        assert response.status_code == 200


@pytest.mark.asyncio
async def test_delete_node_404(client, mock_node_store):
    mock_node_store.delete.return_value = False

    with patch("src.routers.nodes.get_node_store", return_value=mock_node_store):
        response = client.delete("/nodes/nonexistent:v1.0.0?tenant_id=test-tenant")
        assert response.status_code == 404
