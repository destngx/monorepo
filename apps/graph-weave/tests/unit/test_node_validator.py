import pytest
from src.models.node import (
    NodeCreate,
    NodeResponse,
    NodeConfig,
    InputContract,
    OutputContract,
    ContractField,
    Provenance,
)
from src.services.node_validator import NodeValidator


@pytest.fixture
def validator():
    return NodeValidator()


@pytest.fixture
def valid_node():
    return NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
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
    )


@pytest.fixture
def upstream_node():
    return NodeResponse(
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
                ContractField(name="data", type="object", state_path="$.upstream.data")
            ]
        ),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
    )


@pytest.fixture
def downstream_node():
    return NodeResponse(
        tenant_id="test-tenant",
        node_id="downstream:v1.0.0",
        node_name="downstream",
        version="1.0.0",
        name="Downstream",
        type="agent_node",
        description="",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="data", type="object")]
        ),
        output_contract=OutputContract(produced=[]),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
    )


def test_validate_standalone_success(validator, valid_node):
    errors = validator.validate_standalone(valid_node)
    assert errors == []


def test_validate_standalone_missing_config(validator):
    node = NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        config=NodeConfig(),
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
    )
    errors = validator.validate_standalone(node)
    assert len(errors) == 1
    assert "system_prompt" in errors[0]


def test_validate_standalone_missing_output(validator):
    node = NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="input1", type="string")]
        ),
        output_contract=OutputContract(produced=[]),
    )
    errors = validator.validate_standalone(node)
    assert len(errors) == 1
    assert "produced outputs" in errors[0]


def test_validate_standalone_missing_state_path(validator):
    node = NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="input1", type="string")]
        ),
        output_contract=OutputContract(
            produced=[ContractField(name="output1", type="string")]
        ),
    )
    errors = validator.validate_standalone(node)
    assert len(errors) == 1
    assert "state_path" in errors[0]


def test_validate_standalone_missing_type(validator):
    node = NodeCreate(
        tenant_id="test-tenant",
        node_id="test_node:v1.0.0",
        node_name="test_node",
        version="1.0.0",
        name="Test Node",
        type="agent_node",
        config=NodeConfig(system_prompt="Test"),
        input_contract=InputContract(
            required=[ContractField(name="input1", type="")]
        ),
        output_contract=OutputContract(
            produced=[
                ContractField(
                    name="output1", type="string", state_path="$.test.output1"
                )
            ]
        ),
    )
    errors = validator.validate_standalone(node)
    assert len(errors) == 1
    assert "type" in errors[0]


def test_validate_contract_compatibility_success(
    validator, upstream_node, downstream_node
):
    errors = validator.validate_contract_compatibility(upstream_node, downstream_node)
    assert errors == []


def test_validate_contract_missing_field(validator, upstream_node):
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
            required=[ContractField(name="missing_field", type="string")]
        ),
        output_contract=OutputContract(produced=[]),
        tags=[],
        owner="system",
        status="active",
        reuse_eligible=True,
        provenance=Provenance(),
    )
    errors = validator.validate_contract_compatibility(upstream_node, downstream)
    assert len(errors) == 1
    assert "missing_field" in errors[0]


def test_validate_contract_type_mismatch(validator, downstream_node):
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
    )
    errors = validator.validate_contract_compatibility(upstream, downstream_node)
    assert len(errors) == 1
    assert "Type mismatch" in errors[0]


def test_validate_compositional_workflow_success(validator):
    workflow = {
        "nodes": [
            {"alias": "node1", "node_id": "node1:v1.0.0"},
            {"alias": "node2", "node_id": "node2:v1.0.0"},
        ],
        "edges": [
            {"from": "entry", "to": "node1"},
            {"from": "node1", "to": "node2"},
            {"from": "node2", "to": "exit"},
        ],
    }
    errors = validator.validate_compositional_workflow(workflow)
    assert errors == []


def test_validate_compositional_workflow_unknown_alias(validator):
    workflow = {
        "nodes": [{"alias": "node1", "node_id": "node1:v1.0.0"}],
        "edges": [{"from": "entry", "to": "unknown"}],
    }
    errors = validator.validate_compositional_workflow(workflow)
    assert len(errors) == 1
    assert "unknown alias" in errors[0]


def test_validate_compositional_workflow_edge_from_unknown(validator):
    workflow = {
        "nodes": [{"alias": "node1", "node_id": "node1:v1.0.0"}],
        "edges": [{"from": "unknown", "to": "node1"}],
    }
    errors = validator.validate_compositional_workflow(workflow)
    assert len(errors) == 1
    assert "from" in errors[0]


def test_validate_compositional_workflow_edge_to_unknown(validator):
    workflow = {
        "nodes": [{"alias": "node1", "node_id": "node1:v1.0.0"}],
        "edges": [{"from": "node1", "to": "unknown"}],
    }
    errors = validator.validate_compositional_workflow(workflow)
    assert len(errors) == 1
    assert "to" in errors[0]
