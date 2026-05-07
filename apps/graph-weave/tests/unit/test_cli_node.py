import pytest
import json
from unittest.mock import MagicMock
from src.adapters.langgraph.cli_node import CLINodeHandler
from src.adapters.langgraph.real_executor import RealLangGraphExecutor

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
        "bash", {"command": "echo '{\"foo\": \"bar\"}'", "cwd": None}
    )

def test_cli_node_handler_interpolation(mock_executor):
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "test_cli",
        "type": "cli_node",
        "config": {
            "command": "ls {dir_name}",
            "cwd": "{work_dir}"
        }
    }
    state = {
        "workflow_state": {
            "dir_name": "tmp",
            "work_dir": "/projects"
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
        "bash", {"command": "ls tmp", "cwd": "/projects"}
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
