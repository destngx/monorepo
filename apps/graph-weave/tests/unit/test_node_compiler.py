import pytest
from unittest.mock import AsyncMock, patch
from src.models.node import (
    NodeCreate,
    NodeResponse,
    NodeConfig,
    InputContract,
    OutputContract,
    ContractField,
    Provenance,
)
from src.adapters.node import RedisNodeStore
from src.services.node_compiler import WorkflowCompiler, WorkflowFormatError, WorkflowCompileError


@pytest.fixture
def mock_node_store():
    store = AsyncMock(spec=RedisNodeStore)
    return store


@pytest.fixture
def compiler(mock_node_store):
    return WorkflowCompiler(mock_node_store)


@pytest.fixture
def sample_stored_node():
    return NodeResponse(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        description="A test node",
        config=NodeConfig(
            system_prompt="Test prompt",
            user_prompt_template="Test template",
            input_mapping={"input1": "$.entry.input1"},
            output_key="test_output",
        ),
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
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
        created_at="2026-05-14T00:00:00Z",
        immutable_fields=["config", "input_contract", "output_contract", "type"],
    )


@pytest.mark.asyncio
async def test_compile_compositional_workflow(
    compiler, mock_node_store, sample_stored_node
):
    mock_node_store.get.return_value = sample_stored_node

    workflow = {
        "nodes": [
            {"id": "entry", "type": "entry"},
            {
                "alias": "test",
                "node_id": "test_node:v1.0.0",
                "overrides": {"input_mapping": {"input1": "$.entry.custom"}},
            },
            {"id": "exit", "type": "exit"},
        ],
        "edges": [
            {"from": "entry", "to": "test"},
            {"from": "test", "to": "exit"},
        ],
    }

    result = await compiler.compile(workflow)

    assert len(result["nodes"]) == 3
    assert result["nodes"][1]["id"] == "test"
    assert result["nodes"][1]["type"] == "agent_node"
    assert result["nodes"][1]["config"]["system_prompt"] == "Test prompt"
    assert result["nodes"][1]["config"]["input_mapping"]["input1"] == "$.entry.custom"


@pytest.mark.asyncio
async def test_compile_rejects_legacy(compiler):
    workflow = {
        "nodes": [{"id": "test", "type": "agent_node", "config": {}}],
        "edges": [],
    }

    with pytest.raises(WorkflowFormatError):
        await compiler.compile(workflow)


@pytest.mark.asyncio
async def test_compile_missing_node_raises_error(compiler, mock_node_store):
    mock_node_store.get.return_value = None

    workflow = {
        "nodes": [{"alias": "test", "node_id": "nonexistent:v1.0.0"}],
        "edges": [],
    }

    with pytest.raises(WorkflowCompileError):
        await compiler.compile(workflow)


@pytest.mark.asyncio
async def test_validate_references_success(compiler, mock_node_store):
    mock_node_store.exists.return_value = True

    workflow = {
        "nodes": [{"alias": "test", "node_id": "test_node:v1.0.0"}],
        "edges": [],
    }

    errors = await compiler.validate_references(workflow)
    assert errors == []


@pytest.mark.asyncio
async def test_validate_references_missing(compiler, mock_node_store):
    mock_node_store.exists.return_value = False

    workflow = {
        "nodes": [{"alias": "test", "node_id": "nonexistent:v1.0.0"}],
        "edges": [],
    }

    errors = await compiler.validate_references(workflow)
    assert len(errors) == 1
    assert "nonexistent:v1.0.0" in errors[0]


@pytest.mark.asyncio
async def test_validate_contracts_success(
    compiler, mock_node_store, sample_stored_node
):
    matching_node = NodeResponse(
        tenant_id="test-tenant",
        node_id="matching:v1.0.0",
        node_name="matching",
        version="1.0.0",
        name="Matching Node",
        type="agent_node",
        description="",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="output1", type="string")]
        ),
        output_contract=OutputContract(
            produced=[
                ContractField(name="output1", type="string", state_path="$.matching.output1")
            ]
        ),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
        created_at="",
        immutable_fields=[],
    )

    async def mock_get(node_id):
        return sample_stored_node if node_id == "test_node:v1.0.0" else matching_node

    mock_node_store.get.side_effect = mock_get

    workflow = {
        "nodes": [
            {"alias": "node1", "node_id": "test_node:v1.0.0"},
            {"alias": "node2", "node_id": "matching:v1.0.0"},
        ],
        "edges": [{"from": "node1", "to": "node2"}],
    }

    errors = await compiler.validate_contracts(workflow)
    assert errors == []


@pytest.mark.asyncio
async def test_validate_contracts_incompatible(compiler, mock_node_store):
    upstream = NodeResponse(
        tenant_id="test-tenant",
        node_id="upstream:v1.0.0",
        node_name="upstream",
        version="1.0.0",
        name="Upstream",
        type="agent_node",
        description="",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(required=[]),
        output_contract=OutputContract(
            produced=[
                ContractField(name="data", type="string", state_path="$.upstream.data")
            ]
        ),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
        created_at="",
        immutable_fields=[],
    )

    downstream = NodeResponse(
        tenant_id="test-tenant",
        node_id="downstream:v1.0.0",
        node_name="downstream",
        version="1.0.0",
        name="Downstream",
        type="agent_node",
        description="",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="missing", type="string")]
        ),
        output_contract=OutputContract(produced=[]),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
        created_at="",
        immutable_fields=[],
    )

    async def mock_get(node_id):
        if node_id == "upstream:v1.0.0":
            return upstream
        elif node_id == "downstream:v1.0.0":
            return downstream
        return None

    mock_node_store.get.side_effect = mock_get

    workflow = {
        "nodes": [
            {"alias": "up", "node_id": "upstream:v1.0.0"},
            {"alias": "down", "node_id": "downstream:v1.0.0"},
        ],
        "edges": [{"from": "up", "to": "down"}],
    }

    errors = await compiler.validate_contracts(workflow)
    assert len(errors) == 1
    assert "missing" in errors[0]
