class MCPRouterError(Exception):
    """Base exception for MCP router errors."""
    pass

class ProviderConfigError(MCPRouterError):
    """Provider configuration error."""
    pass

class ToolExecutionError(MCPRouterError):
    """Tool execution error."""
    pass
