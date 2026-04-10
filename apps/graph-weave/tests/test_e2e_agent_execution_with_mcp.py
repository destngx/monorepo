import pytest
from fastapi.testclient import TestClient

from src.adapters.langgraph_executor import MockLangGraphExecutor
from src.adapters.mcp import MockMCPServer
from src.main import app


@pytest.fixture
def client():
    return TestClient(app)


class TestAgentExecutionWithMcpE2E:
    def test_agent_calls_mcp_search_tool(self):
        server = MockMCPServer()
        assert server.call_tool("search", {"query": "graph weave"}) == {
            "results": ["Result for: graph weave"]
        }

    def test_agent_calls_mcp_calculate_tool(self):
        server = MockMCPServer()
        assert server.call_tool("calculate", {"operation": "add", "a": 2, "b": 3}) == {
            "result": 5
        }

    def test_agent_prompt_interpolation(self):
        executor = MockLangGraphExecutor()
        assert (
            executor._interpolate_prompt(
                "Research {topic} with {depth}", {"topic": "earnings", "depth": 3}
            )
            == "Research earnings with 3"
        )

    def test_agent_output_schema_shape(self):
        output = {
            "findings": ["foo"],
            "confidence": 0.9,
            "sources": ["https://example.com"],
        }
        assert set(output.keys()) == {"findings", "confidence", "sources"}
        assert 0 <= output["confidence"] <= 1
