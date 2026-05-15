import pytest
from unittest.mock import AsyncMock, MagicMock, patch
from src.models.node import NodeCreate, NodeConfig, InputContract, OutputContract, ContractField
from src.adapters.node import RedisNodeStore, ConflictError, NotFoundError


@pytest.fixture
def mock_redis():
    redis = AsyncMock()
    redis.exists.return_value = False
    redis.hset.return_value = 1
    redis.sadd.return_value = 1
    redis.smembers.return_value = set()
    redis.sinter.return_value = set()
    redis.srem.return_value = 1
    redis.delete.return_value = True
    redis.hget.return_value = None
    return redis


@pytest.fixture
def node_store(mock_redis):
    return RedisNodeStore(mock_redis)


@pytest.fixture
def sample_node():
    return NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        description="A test node",
        config=NodeConfig(system_prompt="Test prompt"),
        input_contract=InputContract(
            required=[ContractField(name="input1", type="string")]
        ),
        output_contract=OutputContract(
            produced=[
                ContractField(
                    name="output1", type="string", state_path="$.test.output1"
                )
            ]
        ),
        tags=["test"],
    )


@pytest.mark.asyncio
async def test_create_node_success(node_store, mock_redis, sample_node):
    result = await node_store.create(sample_node)
    assert result.node_id == "test_node:v1.0.0"
    assert result.tenant_id == "test-tenant"
    mock_redis.hset.assert_called_once()
    assert mock_redis.sadd.call_count >= 3


@pytest.mark.asyncio
async def test_create_node_duplicate_raises_conflict(node_store, mock_redis, sample_node):
    mock_redis.exists.return_value = True
    with pytest.raises(ConflictError):
        await node_store.create(sample_node)


@pytest.mark.asyncio
async def test_get_node_exists(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget.return_value = node_dict
    result = await node_store.get("test_node:v1.0.0")
    assert result is not None
    assert result.node_id == "test_node:v1.0.0"


@pytest.mark.asyncio
async def test_get_node_not_exists_returns_none(node_store, mock_redis):
    mock_redis.hget.return_value = None
    result = await node_store.get("nonexistent:v1.0.0")
    assert result is None


@pytest.mark.asyncio
async def test_list_nodes_all(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers.return_value = {"test_node:v1.0.0"}
    mock_redis.hget.return_value = node_dict
    result = await node_store.list()
    assert result.total == 1
    assert result.nodes[0].node_id == "test_node:v1.0.0"


@pytest.mark.asyncio
async def test_list_nodes_by_tags(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.sinter.return_value = {"test_node:v1.0.0"}
    mock_redis.hget.return_value = node_dict
    result = await node_store.list(tags=["test"])
    assert result.total == 1


@pytest.mark.asyncio
async def test_list_nodes_by_name(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers.return_value = {"test_node:v1.0.0"}
    mock_redis.hget.return_value = node_dict
    result = await node_store.list(node_name="test_node")
    assert result.total == 1


@pytest.mark.asyncio
async def test_update_node_metadata_only(node_store, mock_redis, sample_node):
    from src.models.node import NodeUpdate
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget.return_value = node_dict
    mock_redis.exists.return_value = True
    update = NodeUpdate(name="Updated Name")
    result = await node_store.update("test_node:v1.0.0", update)
    assert result.name == "Updated Name"


@pytest.mark.asyncio
async def test_update_node_not_found_raises_error(node_store, mock_redis):
    from src.models.node import NodeUpdate
    mock_redis.hget.return_value = None
    update = NodeUpdate(name="Updated Name")
    with pytest.raises(NotFoundError):
        await node_store.update("nonexistent:v1.0.0", update)


@pytest.mark.asyncio
async def test_delete_node_removes_indexes(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget.return_value = node_dict
    result = await node_store.delete("test_node:v1.0.0")
    assert result is True
    assert mock_redis.srem.call_count >= 3
    mock_redis.delete.assert_called_once()


@pytest.mark.asyncio
async def test_delete_node_not_exists_returns_false(node_store, mock_redis):
    mock_redis.hget.return_value = None
    result = await node_store.delete("nonexistent:v1.0.0")
    assert result is False


@pytest.mark.asyncio
async def test_exists_returns_true(node_store, mock_redis):
    mock_redis.exists.return_value = True
    result = await node_store.exists("test_node:v1.0.0")
    assert result is True


@pytest.mark.asyncio
async def test_exists_returns_false(node_store, mock_redis):
    mock_redis.exists.return_value = False
    result = await node_store.exists("nonexistent:v1.0.0")
    assert result is False


@pytest.mark.asyncio
async def test_find_by_name_returns_all_versions(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.dict()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers.return_value = {"test_node:v1.0.0"}
    mock_redis.hget.return_value = node_dict
    result = await node_store.find_by_name("test_node")
    assert len(result) == 1
    assert result[0].node_id == "test_node:v1.0.0"
