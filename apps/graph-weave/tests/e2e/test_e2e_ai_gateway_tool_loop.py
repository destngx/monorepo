import pytest
from unittest.mock import patch, MagicMock
import json
from src.adapters.langgraph_executor import RealLangGraphExecutor
from src.adapters.mcp_router import MCPRouter

class TestAIGatewayToolLoop:
    """
    E2E-style test for the AI Gateway Tool loop (ping-pong).
    Verifies that the RealLangGraphExecutor correctly handles multi-step tool calls.
    """

    @patch("src.adapters.ai_gateway_adapter.httpx.Client.post")
    def test_full_tool_loop_execution(self, mock_post):
        # Setup: Mock the two-step interaction with the AI Gateway
        
        # turn 1: Assistant requests the 'search' tool
        resp1 = MagicMock()
        resp1.status_code = 200
        resp1.json.return_value = {
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": None,
                    "tool_calls": [{
                        "id": "call_abc_123",
                        "type": "function",
                        "function": {
                            "name": "search",
                            "arguments": '{"query": "GraphWeave architecture"}'
                        }
                    }]
                }
            }],
            "usage": {"total_tokens": 150}
        }
        
        # turn 2: Assistant synthesizes the final answer after tool result is submitted
        resp2 = MagicMock()
        resp2.status_code = 200
        resp2.json.return_value = {
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": '{"analysis": "GraphWeave uses a modular adapter pattern.", "confidence": 0.95}'
                }
            }],
            "usage": {"total_tokens": 100}
        }
        
        mock_post.side_effect = [resp1, resp2]
        
        # Initialize components
        router = MCPRouter()
        executor = RealLangGraphExecutor(mcp_router=router)
        
        # Define a test node
        test_node = {
            "id": "research_agent",
            "type": "agent_node",
            "config": {
                "system_prompt": "You are a technical analyst.",
                "user_prompt_template": "Analyze {topic}",
                "provider": "github-copilot",
                "model": "gpt-4.1",
                "tools": ["search"]
            }
        }
        
        # State with interpolation data
        state = {
            "topic": "GraphWeave architecture",
            "input": {"topic": "GraphWeave architecture"},
            "workflow_state": {}
        }
        
        # Execute the agent node
        result = executor._execute_agent_node(
            run_id="test_run_001",
            node=test_node,
            state=state,
            workflow={"workflow_id": "test_wf_001"}
        )
        
        # Validations
        assert result["research_agent_status"] == "completed"
        assert result["research_agent_output"]["analysis"] == "GraphWeave uses a modular adapter pattern."
        assert result["turns"] == 2
        assert result["tokens_used"] == 250
        
        # Verify Gateway calls
        assert mock_post.call_count == 2
        
        # Inspect second call to ensure tool result was included in history
        second_call_payload = mock_post.call_args_list[1].kwargs["json"]
        messages = second_call_payload["messages"]
        
        # Should have: system, user, assistant(tool_call), tool(result)
        assert len(messages) == 4
        assert messages[2]["role"] == "assistant"
        assert messages[2]["tool_calls"][0]["id"] == "call_abc_123"
        assert messages[3]["role"] == "tool"
        assert messages[3]["tool_call_id"] == "call_abc_123"
        assert "results" in json.loads(messages[3]["content"])
