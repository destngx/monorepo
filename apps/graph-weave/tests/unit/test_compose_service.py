import asyncio
import pytest
from src.services.compose_service import WorkflowComposeService
from src.models.node import NodeResponse


class MockNodeStore:
    """Mock Node Store for composition testing."""
    def __init__(self, nodes):
        self.nodes = nodes
        
    async def get(self, node_id: str):
        if node_id in self.nodes:
            return NodeResponse(**self.nodes[node_id])
        return None


def test_workflow_compose_service_success(monkeypatch):
    # Mock node definitions representing dynamic catalog nodes
    mock_nodes = {
        "nodes:system:node_a:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:node_a:v1.0.0",
            "node_name": "node_a",
            "version": "1.0.0",
            "name": "Node A",
            "type": "agent_node",
            "description": "Agent A",
            "config": {
                "system_prompt": "You are agent A. Your output MUST be raw JSON ONLY.",
                "user_prompt_template": "Process {file_content}"
            },
            "input_contract": {
                "required": [{"name": "file_content", "type": "string", "description": "file content", "state_path": "$.input.file_content"}],
                "optional": []
            },
            "output_contract": {
                "produced": [{"name": "sources_markdown", "type": "string", "description": "sources MD", "state_path": "$.node_a.sources_markdown"}]
            },
            "capabilities": [],
            "tags": [],
            "owner": "system",
            "status": "active",
            "reuse_eligible": True,
            "provenance": {}
        },
        "nodes:system:node_b:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:node_b:v1.0.0",
            "node_name": "node_b",
            "version": "1.0.0",
            "name": "Node B",
            "type": "cli_node",
            "description": "CLI B",
            "config": {
                "command": "echo {sources_markdown_shell}"
            },
            "input_contract": {
                "required": [{"name": "sources_markdown", "type": "string", "description": "sources MD", "state_path": "$.node_a.sources_markdown"}],
                "optional": []
            },
            "output_contract": {
                "produced": []
            },
            "capabilities": [],
            "tags": [],
            "owner": "system",
            "status": "active",
            "reuse_eligible": True,
            "provenance": {}
        }
    }
    
    mock_store = MockNodeStore(mock_nodes)
    monkeypatch.setattr("src.services.compose_service.get_node_store", lambda: mock_store)
    
    skeleton = {
        "name": "test-composed-workflow",
        "version": "1.0.0",
        "steps": [
            {"id": "entry", "type": "entry"},
            {"id": "step_a", "type": "agent_node"},
            {"id": "step_b", "type": "cli_node"},
            {"id": "exit", "type": "exit"}
        ],
        "edges": [
            {"from": "entry", "to": "step_a"},
            {"from": "step_a", "to": "step_b"},
            {"from": "step_b", "to": "exit"}
        ],
        "entry_point": "entry",
        "exit_point": "exit"
    }
    
    node_map = {
        "step_a": "nodes:system:node_a:v1.0.0",
        "step_b": "nodes:system:node_b:v1.0.0"
    }
    
    composed = asyncio.run(WorkflowComposeService.compose("system", skeleton, node_map))
    
    assert composed["name"] == "test-composed-workflow"
    assert composed["version"] == "1.0.0"
    assert len(composed["nodes"]) == 4
    
    # step_a should map 'file_content' to Entry global input $.input.file_content
    step_a_node = next(n for n in composed["nodes"] if n["id"] == "step_a")
    assert step_a_node["overrides"]["input_mapping"]["file_content"] == "$.input.file_content"
    
    # step_b should map 'sources_markdown' to step_a's produced result ($.nodes.step_a.result.sources_markdown)
    step_b_node = next(n for n in composed["nodes"] if n["id"] == "step_b")
    assert step_b_node["overrides"]["input_mapping"]["sources_markdown"] == "$.nodes.step_a.result.sources_markdown"
    
    # exit node output_mapping should map sources_markdown to $.nodes.step_a.result.sources_markdown
    exit_node = next(n for n in composed["nodes"] if n["id"] == "exit")
    assert exit_node["config"]["output_mapping"]["sources_markdown"] == "$.nodes.step_a.result.sources_markdown"
