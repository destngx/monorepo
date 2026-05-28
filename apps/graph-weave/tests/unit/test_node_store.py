import asyncio
import pytest
from unittest.mock import Mock
from src.models.node import NodeCreate, NodeConfig, InputContract, OutputContract, ContractField
from src.adapters.node import RedisNodeStore, ConflictError, NotFoundError


class FakeRedis:
    def __init__(self):
        self.exists_result = False
        self.hset_result = 1
        self.sadd_result = 1
        self.smembers_result = set()
        self.sinter_result = set()
        self.srem_result = 1
        self.delete_result = True
        self.hget_result = None

        self.exists_mock = Mock()
        self.hset_mock = Mock()
        self.sadd_mock = Mock()
        self.smembers_mock = Mock()
        self.sinter_mock = Mock()
        self.srem_mock = Mock()
        self.delete_mock = Mock()
        self.hget_mock = Mock()

    def node_key(self, node_id):
        return f"nodes:default:{node_id}"

    def node_index_key(self):
        return "nodes:default:index"

    def node_name_key(self, node_name):
        return f"nodes:default:name:{node_name}"

    def node_tag_key(self, tag):
        return f"nodes:default:tags:{tag}"

    async def exists(self, key):
        self.exists_mock(key)
        return self.exists_result

    async def hset(self, key, field, value):
        self.hset_mock(key, field=field, value=value)
        return self.hset_result

    async def sadd(self, key, *values):
        self.sadd_mock(key, *values)
        return self.sadd_result

    async def smembers(self, key):
        self.smembers_mock(key)
        return self.smembers_result

    async def sinter(self, *keys):
        self.sinter_mock(*keys)
        return self.sinter_result

    async def srem(self, key, *values):
        self.srem_mock(key, *values)
        return self.srem_result

    async def delete(self, key):
        self.delete_mock(key)
        return self.delete_result

    async def hget(self, key, field):
        self.hget_mock(key, field)
        return self.hget_result


class SyncFakeRedis(FakeRedis):
    def exists(self, key):
        self.exists_mock(key)
        return self.exists_result

    def hset(self, key, field, value):
        self.hset_mock(key, field=field, value=value)
        return self.hset_result

    def sadd(self, key, *values):
        self.sadd_mock(key, *values)
        return self.sadd_result

    def smembers(self, key):
        self.smembers_mock(key)
        return self.smembers_result

    def sinter(self, *keys):
        self.sinter_mock(*keys)
        return self.sinter_result

    def srem(self, key, *values):
        self.srem_mock(key, *values)
        return self.srem_result

    def delete(self, key):
        self.delete_mock(key)
        return self.delete_result

    def hget(self, key, field):
        self.hget_mock(key, field)
        return self.hget_result


@pytest.fixture
def mock_redis():
    return FakeRedis()


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


def test_create_node_success(node_store, mock_redis, sample_node):
    result = asyncio.run(node_store.create(sample_node))
    assert result.node_id == "test_node:v1.0.0"
    assert result.tenant_id == "test-tenant"
    mock_redis.hset_mock.assert_called_once()
    assert mock_redis.sadd_mock.call_count >= 3


def test_create_node_duplicate_raises_conflict(node_store, mock_redis, sample_node):
    mock_redis.exists_result = True
    with pytest.raises(ConflictError):
        asyncio.run(node_store.create(sample_node))


def test_get_node_exists(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.get("test_node:v1.0.0"))
    assert result is not None
    assert result.node_id == "test_node:v1.0.0"


def test_get_node_not_exists_returns_none(node_store, mock_redis):
    mock_redis.hget_result = None
    result = asyncio.run(node_store.get("nonexistent:v1.0.0"))
    assert result is None


def test_list_nodes_all(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers_result = {"test_node:v1.0.0"}
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.list())
    assert result.total == 1
    assert result.nodes[0].node_id == "test_node:v1.0.0"


def test_list_nodes_by_tags(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.sinter_result = {"test_node:v1.0.0"}
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.list(tags=["test"]))
    assert result.total == 1


def test_list_nodes_by_name(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers_result = {"test_node:v1.0.0"}
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.list(node_name="test_node"))
    assert result.total == 1


def test_update_node_metadata_only(node_store, mock_redis, sample_node):
    from src.models.node import NodeUpdate
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget_result = node_dict
    mock_redis.exists_result = True
    update = NodeUpdate(name="Updated Name")
    result = asyncio.run(node_store.update("test_node:v1.0.0", update))
    assert result.name == "Updated Name"


def test_update_node_config_and_prompts(node_store, mock_redis, sample_node):
    from src.models.node import NodeUpdate, NodeConfig
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget_result = node_dict
    mock_redis.exists_result = True
    new_config = NodeConfig(system_prompt="Brand new system instruction prompt", user_prompt_template="Analyze query: {query}")
    update = NodeUpdate(config=new_config)
    result = asyncio.run(node_store.update("test_node:v1.0.0", update))
    assert result.config.system_prompt == "Brand new system instruction prompt"
    assert result.config.user_prompt_template == "Analyze query: {query}"


def test_update_node_not_found_raises_error(node_store, mock_redis):
    from src.models.node import NodeUpdate
    mock_redis.hget_result = None
    update = NodeUpdate(name="Updated Name")
    with pytest.raises(NotFoundError):
        asyncio.run(node_store.update("nonexistent:v1.0.0", update))


def test_delete_node_removes_indexes(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.delete("test_node:v1.0.0"))
    assert result is True
    assert mock_redis.srem_mock.call_count >= 3
    mock_redis.delete_mock.assert_called_once()


def test_delete_node_not_exists_returns_false(node_store, mock_redis):
    mock_redis.hget_result = None
    result = asyncio.run(node_store.delete("nonexistent:v1.0.0"))
    assert result is False


def test_exists_returns_true(node_store, mock_redis):
    mock_redis.exists_result = True
    result = asyncio.run(node_store.exists("test_node:v1.0.0"))
    assert result is True


def test_exists_returns_false(node_store, mock_redis):
    mock_redis.exists_result = False
    result = asyncio.run(node_store.exists("nonexistent:v1.0.0"))
    assert result is False


def test_find_by_name_returns_all_versions(node_store, mock_redis, sample_node):
    import json
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    mock_redis.smembers_result = {"test_node:v1.0.0"}
    mock_redis.hget_result = node_dict
    result = asyncio.run(node_store.find_by_name("test_node"))
    assert len(result) == 1
    assert result[0].node_id == "test_node:v1.0.0"


def test_list_nodes_supports_sync_redis_adapter(sample_node):
    redis = SyncFakeRedis()
    node_store = RedisNodeStore(redis)
    node_dict = sample_node.model_dump()
    node_dict["created_at"] = ""
    node_dict["immutable_fields"] = ["config", "input_contract", "output_contract", "type"]
    redis.smembers_result = {"test_node:v1.0.0"}
    redis.hget_result = node_dict

    result = asyncio.run(node_store.list())

    assert result.total == 1
    assert result.nodes[0].node_id == "test_node:v1.0.0"
