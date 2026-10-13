package domain

// ToolDefinition describes one MCP tool: its name, purpose and JSON Schema input.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Backend     string         `json:"backend"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// ToolResult is the encoded output of one MCP tool call.
type ToolResult struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Content string `json:"content"`
}
