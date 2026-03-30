package mcp

import (
	"apps/wealth-management-engine/port"
	"apps/wealth-management-engine/service"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Server struct {
	healthService port.HealthService
}

func NewServer() *Server {
	return &Server{
		healthService: service.NewHealthService(),
	}
}

func (s *Server) Run(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		response, shouldWrite := s.Handle(line)
		if !shouldWrite {
			continue
		}

		if _, err := fmt.Fprintln(w, string(response)); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func (s *Server) Handle(raw []byte) ([]byte, bool) {
	var request jsonRPCRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		payload, _ := json.Marshal(newErrorResponse(nil, -32700, "Parse error"))
		return payload, true
	}

	if request.ID == nil {
		return nil, false
	}

	switch request.Method {
	case "initialize":
		return mustMarshal(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: map[string]any{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]any{
					"name":    "wealth-management-engine",
					"version": "0.1.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{},
				},
			},
		}), true
	case "tools/list":
		return mustMarshal(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: map[string]any{
				"tools": []map[string]any{
					{
						"name":        "EngineHealth",
						"description": "Returns engine health status.",
						"inputSchema": map[string]any{
							"type":                 "object",
							"properties":           map[string]any{},
							"additionalProperties": false,
						},
					},
				},
			},
		}), true
	case "tools/call":
		var params struct {
			Name string `json:"name"`
		}
		_ = json.Unmarshal(request.Params, &params)
		if params.Name != "EngineHealth" {
			return mustMarshal(newErrorResponse(request.ID, -32602, "Unknown tool")), true
		}

		health := s.healthService.Check()
		resultJSON, _ := json.Marshal(map[string]any{
			"status":    health.Status,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return mustMarshal(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result: map[string]any{
				"content": []map[string]any{
					{
						"type": "text",
						"text": string(resultJSON),
					},
				},
			},
		}), true
	default:
		return mustMarshal(newErrorResponse(request.ID, -32601, "Method not found")), true
	}
}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func newErrorResponse(id any, code int, message string) jsonRPCResponse {
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
