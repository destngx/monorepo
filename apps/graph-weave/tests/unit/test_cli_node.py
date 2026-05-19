import pytest
import json
from unittest.mock import MagicMock
from src.adapters.langgraph.nodes.cli import CLINodeHandler
from src.adapters.langgraph.runtime.engine import RealLangGraphExecutor

@pytest.fixture
def mock_executor():
    executor = MagicMock(spec=RealLangGraphExecutor)
    executor.mcp_router = MagicMock()
    # Mock interpolation to just return the template or do simple replacement
    def mock_interpolate(template, state, local_context=None):
        ctx = dict(state)
        if local_context:
            ctx.update(local_context)
        import re
        return re.sub(r'\{(\w+)\}', lambda m: str(ctx.get(m.group(1), m.group(0))), template)
    
    executor._interpolate_prompt = mock_interpolate
    executor._get_state_value = lambda path, state: state.get(path.replace("$.", ""))
    return executor

def test_cli_node_handler_execute_success(mock_executor):
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "test_cli",
        "type": "cli_node",
        "config": {
            "command": "echo '{\"foo\": \"bar\"}'",
            "output_key": "cli_out"
        }
    }
    state = {"workflow_state": {}}
    workflow = {}
    
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "success",
        "stdout": '{"foo": "bar"}',
        "stderr": "",
        "exit_code": 0
    }
    
    result = handler.execute("run-1", node, state, workflow)
    
    assert result["node_id"] == "test_cli"
    assert result["status"] == "completed"
    assert result["result"] == {"foo": "bar", "stderr": "", "exit_code": 0, "success": True}
    assert result["cli_out"] == {"foo": "bar", "stderr": "", "exit_code": 0, "success": True}
    
    mock_executor.mcp_router.execute_tool.assert_called_once_with(
        "bash", {"command": "echo '{\"foo\": \"bar\"}'"}
    )

def test_cli_node_handler_interpolation(mock_executor):
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "test_cli",
        "type": "cli_node",
        "config": {
            "command": "ls {dir_name}"
        }
    }
    state = {
        "workflow_state": {
            "dir_name": "tmp"
        }
    }
    workflow = {}
    
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "success",
        "stdout": "file1.txt",
        "stderr": "",
        "exit_code": 0
    }
    
    handler.execute("run-1", node, state, workflow)
    
    mock_executor.mcp_router.execute_tool.assert_called_once_with(
        "bash", {"command": "ls tmp"}
    )

def test_cli_node_handler_failure(mock_executor):
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "test_cli",
        "type": "cli_node",
        "config": {
            "command": "invalid_command"
        }
    }
    state = {"workflow_state": {}}
    workflow = {}
    
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "error",
        "stdout": "",
        "stderr": "command not found",
        "exit_code": 127
    }
    
    result = handler.execute("run-1", node, state, workflow)
    
    assert result["status"] == "failed"
    assert result["exit_code"] == 127
    assert result["result"]["success"] is False

def test_cli_node_handler_non_json_output(mock_executor):
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "test_cli",
        "type": "cli_node",
        "config": {
            "command": "echo hello"
        }
    }
    state = {"workflow_state": {}}
    workflow = {}
    
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "success",
        "stdout": "hello\n",
        "stderr": "",
        "exit_code": 0
    }
    
    result = handler.execute("run-1", node, state, workflow)
    
    assert result["result"]["stdout"] == "hello\n"
    assert "foo" not in result["result"]


def test_cli_node_applies_tool_output_mapping(mock_executor):
    handler = CLINodeHandler(mock_executor)

    node = {
        "id": "create_drafts",
        "type": "cli_node",
        "config": {
            "command": (
                "bash Persona/90_meta/02_scripts/agent/create_drafts_from_json.sh "
                "--source-id src-1 --origin-type note --source-refs card.md "
                "--ideas-json '[]' --summary S --tags t"
            ),
            "tool_output_mapping": {
                "draft_paths": "$.stdout_fields.draft_paths",
                "ideas_count": "$.stdout_fields.draft_count",
                "status": "$.stdout_fields.status",
            },
            "command_contract": {
                "required_command_substrings": [
                    "Persona/90_meta/02_scripts/agent/create_drafts_from_json.sh",
                    "--source-id",
                    "--origin-type",
                    "--source-refs",
                    "--ideas-json",
                    "--summary",
                    "--tags",
                ],
                "forbidden_commands": ["find", "ls", "cat", "grep"],
            },
            "fail_on_error": True,
        },
    }
    state = {"workflow_state": {}}
    workflow = {}

    mock_executor.mcp_router.execute_tool.return_value = {
        "tool": "bash",
        "status": "success",
        "stdout": "STATUS: processed\nDRAFT_COUNT: 2\nDRAFT_PATHS: /a.md,/b.md\n",
        "stderr": "",
        "exit_code": 0,
    }

    result = handler.execute("run-1", node, state, workflow)

    assert result["status"] == "completed"
    assert result["result"]["status"] == "processed"
    assert result["result"]["ideas_count"] == "2"
    assert result["result"]["draft_paths"] == "/a.md,/b.md"


def test_cli_node_fail_on_error_rejects_missing_mapped_output(mock_executor):
    handler = CLINodeHandler(mock_executor)

    node = {
        "id": "create_drafts",
        "type": "cli_node",
        "config": {
            "command": "ls Persona/90_meta/02_scripts/agent",
            "tool_output_mapping": {
                "draft_paths": "$.stdout_fields.draft_paths",
            },
            "command_contract": {
                "forbidden_commands": ["find", "ls", "cat", "grep"],
            },
            "fail_on_error": True,
        },
    }
    state = {"workflow_state": {}}
    workflow = {}

    mock_executor.mcp_router.execute_tool.return_value = {
        "tool": "bash",
        "command": "ls Persona/90_meta/02_scripts/agent",
        "status": "success",
        "stdout": "create_drafts_from_json.sh\n",
        "stderr": "",
        "exit_code": 0,
    }

    with pytest.raises(ValueError) as exc:
        handler.execute("run-1", node, state, workflow)

    assert "forbidden command 'ls'" in str(exc.value)


def test_cli_node_validates_generic_command_contract(mock_executor):
    handler = CLINodeHandler(mock_executor)

    node = {
        "id": "script_step",
        "type": "cli_node",
        "config": {
            "command": (
                "bash Persona/90_meta/02_scripts/agent/create_source_card.sh "
                "--source_type note --title T --authors '' --url '' "
                "--summary S --claims '' --tags ''"
            ),
            "command_contract": {
                "forbidden_command_substrings": ["--source_type"],
            },
        },
    }

    with pytest.raises(ValueError) as exc:
        handler.execute("run-1", node, {"workflow_state": {}}, {})

    assert "forbidden substrings: --source_type" in str(exc.value)
    mock_executor.mcp_router.execute_tool.assert_not_called()
