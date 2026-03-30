package mcp

import (
	"encoding/json"
	"testing"
)

func TestGivenInitializeRequestWhenHandleThenReturnsCapabilities(t *testing.T) {
	server := NewServer()
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if !ok {
		t.Fatalf("expected response")
	}

	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}

func TestGivenToolsListWhenHandleThenReturnsEngineHealthTool(t *testing.T) {
	server := NewServer()
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
	if !ok {
		t.Fatalf("expected response")
	}

	var response struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(response.Result.Tools) != 1 || response.Result.Tools[0].Name != "EngineHealth" {
		t.Fatalf("expected EngineHealth tool")
	}
}

func TestGivenToolsCallEngineHealthWhenHandleThenReturnsHealthPayload(t *testing.T) {
	server := NewServer()
	responseRaw, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"EngineHealth","arguments":{}}}`))
	if !ok {
		t.Fatalf("expected response")
	}

	var response map[string]any
	if err := json.Unmarshal(responseRaw, &response); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if response["error"] != nil {
		t.Fatalf("expected no error, got %v", response["error"])
	}
}
