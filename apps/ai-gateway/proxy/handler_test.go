package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/types"
)

// MockProvider implements providers.Provider for testing.
type MockProvider struct {
	name       string
	configured bool
	ready      bool
	chatResp   *types.ChatResponse
}

func (m *MockProvider) Name() string { return m.name }
func (m *MockProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
	if m.chatResp != nil {
		return m.chatResp, nil
	}
	return &types.ChatResponse{
		ID:      "mock-id",
		Object:  "chat.completion",
		Model:   req.Model,
		Choices: []types.Choice{{Message: types.Message{Role: "assistant", Content: "Hello from mock"}}},
	}, nil
}
func (m *MockProvider) ChatStream(ctx context.Context, req types.ChatRequest, w io.Writer) (types.Usage, error) {
	return types.Usage{}, nil
}
func (m *MockProvider) ListModels(ctx context.Context) (*types.ModelsResponse, error) {
	return &types.ModelsResponse{
		Object: "list",
		Data:   []types.ModelInfo{{ID: "mock-model", Object: "model", OwnedBy: "mock"}},
	}, nil
}
func (m *MockProvider) Embeddings(ctx context.Context, req types.EmbeddingRequest) (*types.EmbeddingResponse, error) {
	return &types.EmbeddingResponse{
		Object: "list",
		Model:  req.Model,
		Data: []types.EmbeddingData{
			{Object: "embedding", Index: 0, Embedding: []float32{0.1, 0.2}},
		},
	}, nil
}
func (m *MockProvider) IsConfigured() bool             { return m.configured }
func (m *MockProvider) Ping(ctx context.Context) error { return nil }
func (m *MockProvider) IsReady() bool                  { return m.ready }
func (m *MockProvider) SetReady(r bool)                { m.ready = r }

func TestServeHTTP(t *testing.T) {
	registry := NewRegistry(&config.Config{}) // Empty config

	// Manually register a mock provider
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock
	githubCopilot := &MockProvider{name: "github-copilot", configured: true, ready: true}
	registry.providers["github-copilot"] = githubCopilot

	handler := NewOpenAIHandler(registry)

	t.Run("Unknown Provider", func(t *testing.T) {
		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4.1"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "unknown")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("Unready Provider", func(t *testing.T) {
		githubCopilot.ready = false
		defer func() { githubCopilot.ready = true }()

		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4.1"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("Successful Completion", func(t *testing.T) {
		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4.1"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp types.ChatResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp.ID != "mock-id" && resp.ID != "" {
			t.Errorf("expected mock response ID, got %s", resp.ID)
		}
	})
}

func TestModelsHandler(t *testing.T) {
	registry := NewRegistry(&config.Config{})
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock
	githubCopilot := &MockProvider{name: "github-copilot", configured: true, ready: true}
	registry.providers["github-copilot"] = githubCopilot

	handler := NewModelsHandler(registry)

	t.Run("List Models", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp types.ModelsResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		if len(resp.Data) != 1 || resp.Data[0].ID != "mock-model" {
			t.Errorf("unexpected models response: %+v", resp)
		}
	})
}

func TestEmbeddingsHandler(t *testing.T) {
	registry := NewRegistry(&config.Config{})
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock

	handler := NewEmbeddingsHandler(registry)

	t.Run("Successful Embeddings", func(t *testing.T) {
		reqBody, _ := json.Marshal(types.EmbeddingRequest{
			Model: "text-embedding-3-small",
			Input: "test input",
		})
		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp types.EmbeddingResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		if len(resp.Data) != 1 || len(resp.Data[0].Embedding) != 2 {
			t.Errorf("unexpected embeddings response: %+v", resp)
		}
	})
}

func TestToolCallHandler(t *testing.T) {
	registry := NewRegistry(&config.Config{})
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock
	githubCopilot := &MockProvider{name: "github-copilot", configured: true, ready: true}
	registry.providers["github-copilot"] = githubCopilot

	handler := NewOpenAIHandler(registry)

	t.Run("Tool Call Response", func(t *testing.T) {
		// Mock a tool call response
		githubCopilot.chatResp = &types.ChatResponse{
			Choices: []types.Choice{{
				Message: types.Message{
					Role: "assistant",
					ToolCalls: []types.ToolCall{{
						ID:   "call_123",
						Type: "function",
						Function: types.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"London"}`,
						},
					}},
				},
				FinishReason: "tool_calls",
			}},
		}
		defer func() { githubCopilot.chatResp = nil }()

		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4.1"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp types.ChatResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		if len(resp.Choices[0].Message.ToolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(resp.Choices[0].Message.ToolCalls))
		}
		if resp.Choices[0].FinishReason != "tool_calls" {
			t.Errorf("expected finish_reason tool_calls, got %s", resp.Choices[0].FinishReason)
		}
	})
}
