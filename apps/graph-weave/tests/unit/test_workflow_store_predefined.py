import json

from src.adapters.redis import FallbackStorage, NamespacedRedisClient
from src.adapters.workflow import RedisWorkflowStore, load_workflow_definition
from tests.mocks.redis_mock import MockRedisAdapter


def test_predefined_workflow_get_refreshes_from_resource(monkeypatch, tmp_path):
    resource_dir = tmp_path / "workflows"
    resource_dir.mkdir()
    resource_path = resource_dir / "workflow-generator:v9.9.9.json"

    workflow_id = "workflow-generator:v9.9.9"
    first_definition = {
        "name": "workflow-generator",
        "version": "9.9.9",
        "metadata": {"tags": ["meta"]},
        "nodes": [{"id": "entry", "type": "entry"}],
        "edges": [{"from": "entry", "to": "exit"}],
    }
    second_definition = {
        **first_definition,
        "edges": [{"from": "entry", "to": "validation_gate"}],
    }

    resource_path.write_text(json.dumps(first_definition))
    monkeypatch.setattr(
        "src.adapters.workflow.PREDEFINED_WORKFLOWS",
        {workflow_id: resource_path.name},
    )
    monkeypatch.setattr(
        "src.adapters.workflow.PREDEFINED_WORKFLOWS_DIR",
        str(resource_dir),
    )

    redis_client = NamespacedRedisClient(MockRedisAdapter(), FallbackStorage())
    store = RedisWorkflowStore(redis_client)

    first = store.get("system", workflow_id)
    assert first["definition"]["edges"] == [{"from": "entry", "to": "exit"}]

    resource_path.write_text(json.dumps(second_definition))

    refreshed = store.get("system", workflow_id)
    assert refreshed["definition"]["edges"] == [
        {"from": "entry", "to": "validation_gate"}
    ]


def test_create_node_predefined_workflow_loads_component_manifest():
    definition = load_workflow_definition(
        "/Users/ez2/projects/personal/monorepo/apps/graph-weave/src/resources/workflows/create-node:v1.0.0.json"
    )

    assert definition["name"] == "create-node"
    assert definition["version"] == "1.0.0"
    assert [node["id"] for node in definition["nodes"]] == [
        "entry",
        "config_generator",
        "node_validator",
        "validation_gate",
        "node_assembler",
        "exit",
    ]
