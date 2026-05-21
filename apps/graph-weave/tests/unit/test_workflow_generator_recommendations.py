import json
import os

from src.adapters.workflow import load_workflow_definition


FIXTURE_PATH = os.path.join(
    os.path.dirname(__file__),
    "../../src/resources/workflows/workflow-generator:v1.0.0.json",
)


def load_generator_definition():
    return load_workflow_definition(FIXTURE_PATH)


def test_generator_includes_architecture_advisor_and_publishability_contract():
    definition = load_generator_definition()

    nodes = {node["id"]: node for node in definition["nodes"]}
    assert "architecture_advisor" in nodes

    advisor = nodes["architecture_advisor"]
    advisor_schema = advisor["config"]["output_schema"]
    assert "recommendations" in advisor_schema["properties"]
    assert "publishability" in advisor_schema["properties"]

    assembler = nodes["assembler"]
    assembler_schema = assembler["config"]["output_schema"]
    assert "publishability" in assembler_schema["properties"]
    assert "recommendations" in assembler_schema["properties"]
    assert assembler_schema["properties"]["generated_workflow"]["type"] == ["object", "null"]

    exit_node = nodes["exit"]
    output_mapping = exit_node["config"]["output_mapping"]
    assert output_mapping["recommendations"] == "$.assembly_result.recommendations"
    assert output_mapping["publishability"] == "$.assembly_result.publishability"


def test_generator_routes_through_architecture_advisor():
    definition = load_generator_definition()
    edges = definition["edges"]
    assert {"from": "node_builder", "to": "architecture_advisor"} in edges
    assert {"from": "architecture_advisor", "to": "edge_router"} in edges


def test_componentized_manifest_is_resolved_into_full_definition(tmp_path):
    node_dir = tmp_path / "workflow-generator" / "nodes"
    node_dir.mkdir(parents=True)

    (node_dir / "entry.json").write_text(
        json.dumps(
            {
                "id": "entry",
                "type": "entry",
                "config": {"properties": {"query": {"type": "string"}}},
            }
        )
    )
    (node_dir / "exit.json").write_text(
        json.dumps(
            {
                "id": "exit",
                "type": "exit",
                "config": {"output_mapping": {"result": "$.final.result"}},
            }
        )
    )
    (node_dir / "planner.json").write_text(
        json.dumps(
            {
                "id": "planner",
                "type": "agent_node",
                "config": {"output_key": "plan"},
            }
        )
    )

    manifest_path = tmp_path / "workflow-generator.json"
    manifest_path.write_text(
        json.dumps(
            {
                "name": "workflow-generator",
                "version": "1.0.1",
                "components": {
                    "base_dir": "workflow-generator",
                    "nodes": [
                        "nodes/entry.json",
                        "nodes/planner.json",
                        "nodes/exit.json",
                    ],
                },
                "edges": [
                    {"from": "entry", "to": "planner"},
                    {"from": "planner", "to": "exit"},
                ],
                "entry_point": "entry",
                "exit_point": "exit",
            }
        )
    )

    definition = load_workflow_definition(str(manifest_path))

    assert [node["id"] for node in definition["nodes"]] == ["entry", "planner", "exit"]
    assert definition["edges"][0] == {"from": "entry", "to": "planner"}
