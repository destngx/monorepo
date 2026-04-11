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
}

func (m *MockProvider) Name() string { return m.name }
func (m *MockProvider) Chat(ctx context.Context, req types.ChatRequest) (*types.ChatResponse, error) {
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
func (m *MockProvider) IsConfigured() bool     { return m.configured }
func (m *MockProvider) Ping(ctx context.Context) error { return nil }
func (m *MockProvider) IsReady() bool          { return m.ready }
func (m *MockProvider) SetReady(r bool)         { m.ready = r }

func TestServeHTTP(t *testing.T) {
	registry := NewRegistry(&config.Config{}) // Empty config
	
	// Manually register a mock provider
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock

	handler := NewHandler(registry)

	t.Run("Unknown Provider", func(t *testing.T) {
		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "unknown")
		
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Unready Provider", func(t *testing.T) {
		mock.ready = false
		defer func() { mock.ready = true }()

		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rr.Code)
		}
	})

	t.Run("Successful Completion", func(t *testing.T) {
		reqBody, _ := json.Marshal(types.ChatRequest{Model: "gpt-4"})
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(reqBody))
		req.Header.Set("X-AI-Provider", "mock")

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}

		var resp types.ChatResponse
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp.ID != "mock-id" {
			t.Errorf("expected ID mock-id, got %s", resp.ID)
		}
	})
}

func TestModelsHandler(t *testing.T) {
	registry := NewRegistry(&config.Config{})
	mock := &MockProvider{name: "mock", configured: true, ready: true}
	registry.providers["mock"] = mock

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
