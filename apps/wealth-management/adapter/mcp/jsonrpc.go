package mcp

import (
	"apps/wealth-management/domain"
	"encoding/json"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func newResultResponse(id any, result any) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func newErrorResponse(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: map[string]any{"code": code, "message": message}}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(newErrorResponse(nil, -32603, "encode response: "+err.Error()))
	}
	return b
}

func toMCPTools(definitions []domain.ToolDefinition) []map[string]any {
	tools := make([]map[string]any, 0, len(definitions))
	for _, definition := range definitions {
		tools = append(tools, map[string]any{"name": definition.Name, "description": definition.Description, "inputSchema": definition.InputSchema})
	}
	return tools
}
