import pytest
from src.adapters.mcp import MockMCPServer


@pytest.fixture
def mcp_server():
    return MockMCPServer()


class TestMockMCPIntegration:
    def test_mock_mcp_list_tools(self, mcp_server):
        tools = mcp_server.list_tools()
        assert isinstance(tools, list)
        assert len(tools) > 0

    def test_mock_mcp_tool_has_schema(self, mcp_server):
        tools = mcp_server.list_tools()
        assert all("name" in tool for tool in tools)
        assert all("description" in tool for tool in tools)
        assert all("inputSchema" in tool for tool in tools)

    def test_mock_mcp_call_tool(self, mcp_server):
        result = mcp_server.call_tool("calculate", {"operation": "add", "a": 2, "b": 3})
        assert result is not None

    def test_mock_mcp_call_nonexistent_tool_fails(self, mcp_server):
        with pytest.raises(ValueError):
            mcp_server.call_tool("nonexistent_tool", {})

    def test_mock_mcp_resources(self, mcp_server):
        resources = mcp_server.list_resources()
        assert isinstance(resources, list)

    def test_mock_mcp_deterministic_responses(self, mcp_server):
        result1 = mcp_server.call_tool(
            "calculate", {"operation": "add", "a": 2, "b": 3}
        )
        result2 = mcp_server.call_tool(
            "calculate", {"operation": "add", "a": 2, "b": 3}
        )
        assert result1 == result2
