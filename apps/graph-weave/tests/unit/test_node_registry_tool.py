from src.adapters.mcp.tools.node_registry import NodeRegistryTool, handle_node_registry


class DummyNodeStore:
    pass


class Node:
    def __init__(self, **kwargs):
        self.data = kwargs

    def model_dump(self):
        return self.data


class NodeList:
    def __init__(self, nodes):
        self.nodes = nodes
        self.total = len(nodes)


class InventoryNodeStore:
    async def list(self, **kwargs):
        return NodeList(
            [
                Node(
                    node_id="input_normalizer:v1.0.0",
                    node_name="input_normalizer",
                    name="Input Normalizer",
                    type="agent_node",
                    tags=["input_cleaning", "validation"],
                    description="Normalize inbox input",
                )
            ]
        )


def test_find_compatible_requires_node_name_without_raising_type_error():
    tool = NodeRegistryTool(DummyNodeStore())

    result = handle_node_registry(tool, "find_compatible")

    assert result["tool"] == "node_registry"
    assert result["operation"] == "find_compatible"
    assert result["status"] == "error"
    assert result["error"] == "node_name is required for find_compatible"


def test_list_current_returns_node_inventory():
    tool = NodeRegistryTool(InventoryNodeStore())

    result = handle_node_registry(tool, "list_current")

    assert result["status"] == "success"
    assert result["total"] == 1
    assert result["nodes"][0]["node_id"] == "input_normalizer:v1.0.0"


def test_resolve_steps_matches_existing_and_suggests_missing():
    tool = NodeRegistryTool(InventoryNodeStore())

    result = handle_node_registry(
        tool,
        "resolve_steps",
        steps=[
            {
                "step_id": "step_1_normalize_input",
                "type": "agent_node",
                "capabilities": ["input_cleaning"],
                "description": "Normalize input",
            },
            {
                "step_id": "create_source_cards",
                "type": "cli_node",
                "capabilities": ["cli_execution"],
                "description": "Create source cards",
            },
        ],
    )

    assert result["status"] == "success"
    assert result["nodes"][0]["status"] == "exists"
    assert result["nodes"][0]["node_id"] == "input_normalizer:v1.0.0"
    assert result["nodes"][1]["status"] == "missing"
    assert result["nodes"][1]["suggestion"]["node_name"] == "create_source_cards"
