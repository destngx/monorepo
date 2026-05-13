import pytest
import json
import uuid
from unittest.mock import MagicMock, patch
from src.adapters.langgraph.runtime.engine import RealLangGraphExecutor
from src.services.scheduler_service import SchedulerService

@pytest.fixture
def mock_deps():
    with patch("src.routers.execution.get_executor") as mock_get_executor, \
         patch("src.routers.execution.get_workflow_store") as mock_get_workflow_store, \
         patch("src.routers.execution.status_service.set_status") as mock_set_status, \
         patch("src.routers.execution.status_service.transition_status") as mock_transition_status:
        
        executor = MagicMock(spec=RealLangGraphExecutor)
        mock_get_executor.return_value = executor
        
        workflow_store = MagicMock()
        mock_get_workflow_store.return_value = workflow_store
        
        yield {
            "executor": executor,
            "workflow_store": workflow_store,
            "set_status": mock_set_status,
            "transition_status": mock_transition_status
        }

def test_scheduled_cli_node_execution(mock_deps):
    """
    Test that a scheduled workflow containing a CLI node executes correctly.
    This simulates the full path from scheduler handler to CLI node execution.
    """
    from src.routers.execution import scheduler_execution_handler
    
    tenant_id = "test-tenant"
    workflow_id = "echo-workflow:v1.0.0"
    input_data = {"item_id": "SKU-123"}
    
    # 1. Mock the workflow definition
    workflow_def = {
        "workflow_id": workflow_id,
        "name": "Echo Workflow",
        "version": "1.0.0",
        "definition": {
            "nodes": [
                {"id": "start", "type": "entry"},
                {
                    "id": "run_echo",
                    "type": "cli_node",
                    "config": {
                        "command": "echo 'Process started for {item_id}'",
                        "output_key": "echo_result"
                    }
                },
                {"id": "end", "type": "exit"}
            ],
            "edges": [
                {"source": "start", "target": "run_echo"},
                {"source": "run_echo", "target": "end"}
            ],
            "entry_point": "start",
            "exit_point": "end"
        }
    }
    mock_deps["workflow_store"].get.return_value = workflow_def
    
    # 2. Mock the executor.execute response
    expected_result = {
        "status": "completed",
        "workflow_state": {
            "item_id": "SKU-123",
            "echo_result": {
                "stdout": "Process started for SKU-123\n",
                "exit_code": 0,
                "success": True
            }
        }
    }
    mock_deps["executor"].execute.return_value = expected_result
    
    # 3. Trigger the scheduler handler (this is what APScheduler calls)
    with patch("threading.Thread") as mock_thread:
        scheduler_execution_handler(tenant_id, workflow_id, input_data)
        
        # Verify it queued the execution
        assert mock_deps["set_status"].called
        assert mock_thread.called
        
        # Manually run the background function that the thread would run
        target_func = mock_thread.call_args[1]["target"]
        args = mock_thread.call_args[1]["args"]
        
        target_func(*args)
        
    # 4. Verify the executor was called with the correct workflow and input
    mock_deps["executor"].execute.assert_called_once()
    call_args = mock_deps["executor"].execute.call_args[1]
    assert call_args["workflow"] == workflow_def
    assert call_args["input_data"] == input_data
    
    # Verify status transitions
    assert mock_deps["transition_status"].call_count >= 2 # running -> completed

def test_cli_node_interpolation_in_scheduled_run():
    """
    Specifically test that CLINodeHandler correctly interpolates 
    scheduled input data.
    """
    from src.adapters.langgraph.nodes.cli import CLINodeHandler
    
    mock_executor = MagicMock()
    # Mock interpolation
    def mock_interpolate(template, state, local_context=None):
        ctx = dict(state.get("workflow_state", {}))
        if local_context:
            ctx.update(local_context)
        import re
        return re.sub(r'\{(\w+)\}', lambda m: str(ctx.get(m.group(1), m.group(0))), template)
    
    mock_executor._interpolate_prompt = mock_interpolate
    mock_executor._get_state_value = lambda path, state: state.get("workflow_state", {}).get(path.replace("$.", ""))
    
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "cleanup",
        "type": "cli_node",
        "config": {
            "command": "rm -rf {target_dir}/*.tmp"
        }
    }
    
    # Simulated state from a scheduled run
    state = {
        "workflow_state": {
            "target_dir": "/var/log/app"
        }
    }
    
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "success",
        "stdout": "",
        "exit_code": 0
    }
    
    handler.execute("run-123", node, state, {})
    
    # Verify interpolation worked
    mock_executor.mcp_router.execute_tool.assert_called_once_with(
        "bash", {"command": "rm -rf /var/log/app/*.tmp", "cwd": None}
    )

def test_simple_echo_node():
    """
    The simplest possible test: run an 'echo' command and verify the result.
    """
    from src.adapters.langgraph.nodes.cli import CLINodeHandler
    
    mock_executor = MagicMock()
    # Simple interpolation: just return template as is for this test
    mock_executor._interpolate_prompt = lambda t, s, local_context=None: t
    
    handler = CLINodeHandler(mock_executor)
    
    node = {
        "id": "echo_node",
        "type": "cli_node",
        "config": {
            "command": "echo 'Hello World'",
            "output_key": "greeting"
        }
    }
    
    state = {"workflow_state": {}}
    
    # Mock the tool response
    mock_executor.mcp_router.execute_tool.return_value = {
        "status": "success",
        "stdout": "Hello World\n",
        "stderr": "",
        "exit_code": 0
    }
    
    result = handler.execute("run-simple", node, state, {})
    
    assert result["status"] == "completed"
    assert result["greeting"]["stdout"] == "Hello World\n"
    assert result["greeting"]["success"] is True
    
    mock_executor.mcp_router.execute_tool.assert_called_once_with(
        "bash", {"command": "echo 'Hello World'", "cwd": None}
    )
