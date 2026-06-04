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
    monkeypatch.setattr("src.services.compose_service.get_node_store", lambda *args, **kwargs: mock_store)
    
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
    
    # exit node output_mapping should use wildcard and fallback format
    exit_node = next(n for n in composed["nodes"] if n["id"] == "exit")
    assert exit_node["config"]["output_mapping"]["sources_markdown"] == [
        "$.nodes.*.result.sources_markdown",
        "$.input.sources_markdown"
    ]


def test_workflow_compose_service_coalescing(monkeypatch):
    mock_nodes = {
        "nodes:system:normalize_input:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:normalize_input:v1.0.0",
            "node_name": "normalize_input",
            "version": "1.0.0",
            "name": "Normalize Input",
            "type": "agent_node",
            "description": "normalize",
            "config": {
                "system_prompt": "normalize",
                "user_prompt_template": "normalize"
            },
            "input_contract": {
                "required": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.input.file_content"}],
                "optional": []
            },
            "output_contract": {
                "produced": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.normalize_input.result.file_content"}]
            },
            "capabilities": [],
            "tags": [],
            "owner": "system",
            "status": "active",
            "reuse_eligible": True,
            "provenance": {}
        },
        "nodes:system:fetch_url_content:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:fetch_url_content:v1.0.0",
            "node_name": "fetch_url_content",
            "version": "1.0.0",
            "name": "Fetch URL",
            "type": "agent_node",
            "description": "fetch",
            "config": {
                "system_prompt": "fetch",
                "user_prompt_template": "fetch {file_content}"
            },
            "input_contract": {
                "required": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.normalize_input.result.file_content"}],
                "optional": []
            },
            "output_contract": {
                "produced": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.fetch_url_content.result.file_content"}]
            },
            "capabilities": [],
            "tags": [],
            "owner": "system",
            "status": "active",
            "reuse_eligible": True,
            "provenance": {}
        },
        "nodes:system:process_media:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:process_media:v1.0.0",
            "node_name": "process_media",
            "version": "1.0.0",
            "name": "Process Media",
            "type": "cli_node",
            "description": "media",
            "config": {
                "command": "ocr {file_content_shell}"
            },
            "input_contract": {
                "required": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.normalize_input.result.file_content"}],
                "optional": []
            },
            "output_contract": {
                "produced": [
                    {"name": "file_content", "type": "string", "description": "content", "state_path": "$.process_media.result.file_content"},
                    {"name": "stdout", "type": "string", "description": "stdout", "state_path": "$.process_media.stdout"},
                    {"name": "stderr", "type": "string", "description": "stderr", "state_path": "$.process_media.stderr"},
                    {"name": "exit_code", "type": "number", "description": "exit_code", "state_path": "$.process_media.exit_code"}
                ]
            },
            "capabilities": [],
            "tags": [],
            "owner": "system",
            "status": "active",
            "reuse_eligible": True,
            "provenance": {}
        },
        "nodes:system:extract_and_format:v1.0.0": {
            "tenant_id": "system",
            "node_id": "nodes:system:extract_and_format:v1.0.0",
            "node_name": "extract_and_format",
            "version": "1.0.0",
            "name": "Extract",
            "type": "agent_node",
            "description": "extract",
            "config": {
                "system_prompt": "extract",
                "user_prompt_template": "extract {file_content}"
            },
            "input_contract": {
                "required": [{"name": "file_content", "type": "string", "description": "content", "state_path": "$.normalize_input.result.file_content"}],
                "optional": []
            },
            "output_contract": {
                "produced": [
                    {"name": "sources_markdown", "type": "string", "description": "sources", "state_path": "$.extract_and_format.sources_markdown"},
                    {"name": "drafts_markdown", "type": "string", "description": "drafts", "state_path": "$.extract_and_format.drafts_markdown"}
                ]
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
    monkeypatch.setattr("src.services.compose_service.get_node_store", lambda *args, **kwargs: mock_store)
    
    skeleton = {
        "name": "test-coalesced-workflow",
        "version": "1.0.0",
        "steps": [
            {"id": "entry", "type": "entry"},
            {"id": "normalize_input", "type": "agent_node"},
            {"id": "fetch_url_content", "type": "agent_node"},
            {"id": "process_media", "type": "cli_node"},
            {"id": "extract_and_format", "type": "agent_node"},
            {"id": "exit", "type": "exit"}
        ],
        "edges": [
            {"from": "entry", "to": "normalize_input"},
            {"from": "normalize_input", "to": "fetch_url_content"},
            {"from": "normalize_input", "to": "process_media"},
            {"from": "normalize_input", "to": "extract_and_format"},
            {"from": "fetch_url_content", "to": "process_media"},
            {"from": "fetch_url_content", "to": "extract_and_format"},
            {"from": "process_media", "to": "extract_and_format"},
            {"from": "extract_and_format", "to": "exit"}
        ],
        "entry_point": "entry",
        "exit_point": "exit"
    }
    
    node_map = {
        "normalize_input": "nodes:system:normalize_input:v1.0.0",
        "fetch_url_content": "nodes:system:fetch_url_content:v1.0.0",
        "process_media": "nodes:system:process_media:v1.0.0",
        "extract_and_format": "nodes:system:extract_and_format:v1.0.0"
    }
    
    composed = asyncio.run(WorkflowComposeService.compose("system", skeleton, node_map))
    
    extract_node = next(n for n in composed["nodes"] if n["id"] == "extract_and_format")
    mapping = extract_node["overrides"]["input_mapping"]["file_content"]
    
    # It must be a list of paths in reverse chronological ancestor order
    assert isinstance(mapping, list)
    assert mapping == [
        "$.nodes.process_media.result.file_content",
        "$.nodes.fetch_url_content.result.file_content",
        "$.nodes.normalize_input.result.file_content",
        "$.input.file_content"
    ]

