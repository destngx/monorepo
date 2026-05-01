import os
import pytest
from src.adapters.langgraph_executor import RealLangGraphExecutor
from src.adapters.mcp_router import MCPRouter
from src.models import OrchestratorConfig
from src.modules.orchestrator_react import OrchestratorReAct
from src.adapters.ai_gateway_adapter import AIGatewayClient

def test_e2e_bash_tool_in_orchestrator():
    """
    Test using a real AI Gateway client to simulate the Orchestrator node 
    using the bash tool to create and run a script.
    """
    # Initialize real client pointing to ezmacmini:8080
    client = AIGatewayClient(base_url="http://ezmacmini:8080/v1")
    mcp_router = MCPRouter()
    
    # Ensure tmp dir exists
    tmp_dir = os.path.join(os.getcwd(), "tmp")
    os.makedirs(tmp_dir, exist_ok=True)
    
    events = []
    def emit(event_name, payload):
        events.append((event_name, payload))

    react = OrchestratorReAct(
        client=client,
        mcp_router=mcp_router,
        emit=emit,
        max_context_messages=20
    )

    # Note: Using provider/model that the gateway should handle
    config = OrchestratorConfig(
        system_prompt="You are a developer assistant. Use the bash tool to solve tasks.",
        allowed_skills=["bash"],
        max_iterations=10,
        provider="github-copilot",
        model="gpt-4.1"
    )

    # Task: Create a python script that prints 'E2E Success' and then run it.
    user_prompt = "Create a python script at ./tmp/test_script_e2e.py that prints 'E2E Success', then execute it and report the result."

    result = react.run(
        run_id="test_run_e2e",
        node_id="test_node_e2e",
        config=config,
        workflow_state={},
        user_prompt=user_prompt
    )

    # Verify results
    assert "orchestrator_trace" in result
    trace = result["orchestrator_trace"]
    
    # We expect at least one bash tool call
    bash_calls = [t for t in trace if t.get("type") == "action" and t.get("tool_name") == "bash"]
    assert len(bash_calls) >= 1
    
    # Verify execution output in trace
    observations = [t for t in trace if t.get("type") == "observation" and t.get("tool_name") == "bash"]
    found_success = any("E2E Success" in str(o.get("content", "")) for o in observations)
    assert found_success, "The script execution output 'E2E Success' was not found in the trace."

    # Final result should contain something about success
    final_text = str(result.get("final_result", ""))
    assert "Success" in final_text or "E2E Success" in final_text

    # Clean up
    test_script_path = os.path.join(tmp_dir, "test_script_e2e.py")
    if os.path.exists(test_script_path):
        os.remove(test_script_path)
