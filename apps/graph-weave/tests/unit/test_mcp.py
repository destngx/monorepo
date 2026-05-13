import pytest
from src.adapters.mcp import ToolRegistry


@pytest.fixture
def tool_registry():
    return ToolRegistry()


class TestToolRegistryIntegration:
    def test_registry_list_tools(self, tool_registry):
        tools = tool_registry.list_tools()
        assert isinstance(tools, list)
        assert len(tools) > 0

    def test_registry_tool_has_schema(self, tool_registry):
        tools = tool_registry.list_tools()
        assert all("name" in tool for tool in tools)
        assert all("description" in tool for tool in tools)
        assert all("inputSchema" in tool for tool in tools)

    def test_registry_call_tool(self, tool_registry):
        result = tool_registry.call_tool("calculate", {"operation": "add", "a": 2, "b": 3})
        assert result is not None

    def test_registry_call_nonexistent_tool_fails(self, tool_registry):
        with pytest.raises(ValueError):
            tool_registry.call_tool("nonexistent_tool", {})

    def test_registry_resources(self, tool_registry):
        resources = tool_registry.list_resources()
        assert isinstance(resources, list)

    def test_registry_deterministic_responses(self, tool_registry):
        result1 = tool_registry.call_tool(
            "calculate", {"operation": "add", "a": 2, "b": 3}
        )
        result2 = tool_registry.call_tool(
            "calculate", {"operation": "add", "a": 2, "b": 3}
        )
        assert result1 == result2
