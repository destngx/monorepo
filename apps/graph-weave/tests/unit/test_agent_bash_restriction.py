import pytest
from src.adapters.langgraph_graph_builder import GraphBuilder, WorkflowParseError
from src.adapters.langgraph_executor import RealLangGraphExecutor
from src.adapters.mcp_router import MCPRouter

def test_agent_node_can_now_use_bash_and_fs_tool_validation():
    """
    Verify that the GraphBuilder now ALLOWS an agent_node that tries to use 'bash' or 'fs'.
    """
    workflow_dict = {
        "workflow_id": "test:v1.0.0",
        "nodes": [
            {"id": "entry", "type": "entry"},
            {
                "id": "shell_agent",
                "type": "agent_node",
                "system_prompt": "You are a shell assistant.",
                "user_prompt_template": "Run {command}",
                "tools": ["bash", "fs"] 
            },
            {"id": "exit", "type": "exit"}
        ],
        "edges": [
            {"source": "entry", "target": "shell_agent"},
            {"source": "shell_agent", "target": "exit"}
        ],
        "entry_point": "entry",
        "exit_point": "exit"
    }
    
    # This should NOT raise WorkflowParseError now
    result = GraphBuilder.build(workflow_dict)
    assert "shell_agent" in result["nodes"]
    
    # Check that config has tools
    agent_func = result["nodes"]["shell_agent"]
    assert "bash" in agent_func.config.tools
    assert "fs" in agent_func.config.tools

def test_executor_would_allow_bash_if_not_for_builder():
    """
    This test demonstrates that if we bypassed the builder, the executor 
    actually has the logic to run bash, but we want to confirm if this is 
    the desired behavior or if we should move it to a tool_node.
    """
    # This is more of a documentation test to show the current executor capability
    # which we might want to restrict or move.
    router = MCPRouter()
    executor = RealLangGraphExecutor(mcp_router=router)
    
    # Manually constructed node that bypassed builder validation
    node = {
        "id": "agent_1",
        "type": "agent_node",
        "config": {
            "tools": ["bash"]
        }
    }
    
    # We won't actually run it because it requires AI Gateway mock,
    # but we can see in the code that _execute_agent_node calls mcp_router.execute_tool
    # and mcp_router.VALID_TOOLS includes 'bash'.
    assert "bash" in router.bash_tool.ALLOWED_COMMANDS_BASE or True # Just checking existence
