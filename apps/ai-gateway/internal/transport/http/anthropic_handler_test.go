package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
	"apps/ai-gateway/internal/service"

	"github.com/stretchr/testify/assert"
)

type MockTestProvider struct {
	name                string
	chatCallCount       int
	streamCallCount     int
	lastChatModel       string
	lastReasoningEffort string
	lastMaxTokens       *int
	lastMaxCompletion   *int
}

func (m *MockTestProvider) Name() string { return m.name }
func (m *MockTestProvider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	m.chatCallCount++
	m.lastChatModel = req.Model
	m.lastReasoningEffort = req.ReasoningEffort
	m.lastMaxTokens = req.MaxTokens
	m.lastMaxCompletion = req.MaxCompletionTokens
	return &domain.ChatResponse{}, nil
}
func (m *MockTestProvider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	m.streamCallCount++
	return domain.Usage{}, nil
}
func (m *MockTestProvider) ListModels(context.Context) (*domain.ModelsResponse, error) {
	return &domain.ModelsResponse{
		Object: "list",
		Data: []domain.ModelInfo{
			{ID: "mock-model", Object: "model", OwnedBy: m.name},
		},
	}, nil
}
func (m *MockTestProvider) Embeddings(context.Context, domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, nil
}
func (m *MockTestProvider) IsConfigured() bool                     { return true }
func (m *MockTestProvider) Ping(ctx context.Context) error         { return nil }
func (m *MockTestProvider) Usage(ctx context.Context) (any, error) { return nil, nil }
func (m *MockTestProvider) IsReady() bool                          { return true }
func (m *MockTestProvider) SetReady(bool)                          {}

func setupTestDeps() (*AnthropicHandler, *MockTestProvider, *MockTestProvider) {
	cfg := &config.Config{}
	reg := service.NewRegistry(cfg)

	// Since we don't have true config keys, default mappers might fail or act up if models aren't right,
	// so we force model mapping:
	reg.Mapper = service.NewModelMapper()
	reg.Mapper.DefaultTarget = service.RouteTarget{Provider: domain.ProviderOpenAI}

	mockOpenAI := &MockTestProvider{name: domain.ProviderOpenAI}
	mockAnthropic := &MockTestProvider{name: domain.ProviderAnthropic}

	reg.RegisterForTest(mockOpenAI)
	reg.RegisterForTest(mockAnthropic)

	return NewAnthropicHandler(reg), mockOpenAI, mockAnthropic
}

func setupDefaultRouteTestDeps() (*AnthropicHandler, *MockTestProvider) {
	cfg := &config.Config{}
	reg := service.NewRegistry(cfg)
	reg.Mapper = service.NewModelMapper()

	mockGitHubCopilot := &MockTestProvider{name: domain.ProviderGitHubCopilot}
	reg.RegisterForTest(mockGitHubCopilot)

	return NewAnthropicHandler(reg), mockGitHubCopilot
}

func setupOpenAIRouteTestDeps() (*AnthropicHandler, *MockTestProvider, *MockTestProvider) {
	cfg := &config.Config{AnthropicRoute: "openai-gpt-5.4-mini-low"}
	reg := service.NewRegistry(cfg)
	reg.Mapper = service.NewModelMapper()

	mockOpenAI := &MockTestProvider{name: domain.ProviderOpenAI}
	mockAnthropic := &MockTestProvider{name: domain.ProviderAnthropic}

	reg.RegisterForTest(mockOpenAI)
	reg.RegisterForTest(mockAnthropic)

	return NewAnthropicHandler(reg), mockOpenAI, mockAnthropic
}

func TestNewAnthropicHandler_EnablesDefaultRouteInterceptor(t *testing.T) {
	handler, _, _ := setupTestDeps()

	assert.NotNil(t, handler.routeInterceptor)
}

func doReq(t *testing.T, handler http.Handler, targetProvider, model string, reqBody anthropic.Request) (*httptest.ResponseRecorder, map[string]any) {
	reqBody.Model = model
	j, _ := json.Marshal(reqBody)
	return doRawReq(t, handler, targetProvider, j)
}

func doRawReq(t *testing.T, handler http.Handler, targetProvider string, body []byte) (*httptest.ResponseRecorder, map[string]any) {
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set(domain.HeaderAIProvider, targetProvider)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httpReq)

	var resp map[string]any
	json.NewDecoder(rr.Body).Decode(&resp)
	return rr, resp
}

func assertErrorType(t *testing.T, resp map[string]any, errType string) {
	assert.Equal(t, "error", resp["type"])
	errObj, ok := resp["error"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, errType, errObj["type"])
}

func TestAnthropicHandler_RejectsNativeTools_NonAnthropic(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()

	req := anthropic.Request{
		Tools: []anthropic.Tool{
			{Name: "web_search", Type: "web_search_20250305"},
			{Name: "valid_tool", Type: "function"},
		},
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, resp := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertErrorType(t, resp, "unsupported_native_tool")

	// INVARIANT: Provider must never be called!
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
	assert.Equal(t, 0, mockOpenAI.streamCallCount)
}

func TestAnthropicHandler_RejectsNativeMessageBlock_NonAnthropic(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()

	req := anthropic.Request{
		Messages: []anthropic.Message{
			{
				Role: "assistant",
				Content: []any{
					map[string]any{"type": "server_tool_use", "name": "web_search_20250305", "input": map[string]any{}},
				},
			},
		},
	}

	rr, resp := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertErrorType(t, resp, "unsupported_native_tool")
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
}

func TestAnthropicHandler_RejectsUnmappedToolChoice_NonAnthropic(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()

	req := anthropic.Request{
		Tools: []anthropic.Tool{
			{Name: "valid_tool", Type: "function"},
		},
		ToolChoice: map[string]any{
			"type": "tool",
			"name": "web_search_20241022",
		},
		Messages: []anthropic.Message{{Role: "user", Content: "search this"}},
	}

	rr, resp := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assertErrorType(t, resp, "unsupported_native_tool")
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
}

func TestAnthropicHandler_AllowsNative_AnthropicPassThrough(t *testing.T) {
	handler, _, mockAnthro := setupTestDeps()

	req := anthropic.Request{
		Tools: []anthropic.Tool{
			{Name: "web_search", Type: "web_search_20250305"},
		},
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, domain.ProviderAnthropic, "claude-3-5-sonnet", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockAnthro.chatCallCount)
}

func TestAnthropicHandler_RejectsStream_NonAnthropic(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()

	req := anthropic.Request{
		Stream: true,
		Tools: []anthropic.Tool{
			{Name: "web_search", Type: "web_search_20250305"},
		},
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	req.Model = "gpt-4o"
	j, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewReader(j))
	httpReq.Header.Set(domain.HeaderAIProvider, domain.ProviderOpenAI)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httpReq)

	assert.Equal(t, http.StatusOK, rr.Code) // Streams emit 200 and return event: error
	assert.Contains(t, rr.Body.String(), "event: error")
	assert.Contains(t, rr.Body.String(), "unsupported_native_tool")
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
	assert.Equal(t, 0, mockOpenAI.streamCallCount)
}

func TestAnthropicHandler_RouteInterceptorCanOverrideProviderAndModel(t *testing.T) {
	handler, mockOpenAI, mockAnthro := setupTestDeps()
	handler.SetRouteInterceptor(func(r *http.Request, req anthropic.Request, route AnthropicRoute) (AnthropicRoute, error) {
		route.Provider = mockAnthro
		route.Model = "claude-3-5-sonnet-20241022"
		return route, nil
	})

	req := anthropic.Request{
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
	assert.Equal(t, 1, mockAnthro.chatCallCount)
	assert.Equal(t, "claude-3-5-sonnet-20241022", mockAnthro.lastChatModel)
}

func TestAnthropicHandler_RouteInterceptorErrorStopsProviderCall(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()
	handler.SetRouteInterceptor(func(r *http.Request, req anthropic.Request, route AnthropicRoute) (AnthropicRoute, error) {
		return AnthropicRoute{}, assert.AnError
	})

	req := anthropic.Request{
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
}

func TestAnthropicHandler_DefaultRouteInterceptorUsesGitHubCopilotGPT5MiniHigh(t *testing.T) {
	handler, mockGitHubCopilot := setupDefaultRouteTestDeps()

	req := anthropic.Request{
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, "", "", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockGitHubCopilot.chatCallCount)
	assert.Equal(t, domain.ModelGPT5Mini, mockGitHubCopilot.lastChatModel)
	assert.Equal(t, domain.ReasoningEffortHigh, mockGitHubCopilot.lastReasoningEffort)
}

func TestAnthropicHandler_OpenAIGPT54MiniLowRouteInterceptorExample(t *testing.T) {
	handler, mockOpenAI, mockAnthro := setupOpenAIRouteTestDeps()

	req := anthropic.Request{
		MaxTokens: 128,
		Messages:  []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, domain.ProviderAnthropic, "claude-3-5-sonnet", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockOpenAI.chatCallCount)
	assert.Equal(t, 0, mockAnthro.chatCallCount)
	assert.Equal(t, domain.ModelGPT54Mini, mockOpenAI.lastChatModel)
	assert.Equal(t, domain.ReasoningEffortLow, mockOpenAI.lastReasoningEffort)
	assert.Nil(t, mockOpenAI.lastMaxTokens)
	assert.Nil(t, mockOpenAI.lastMaxCompletion)
}

func TestAnthropicHandler_OpenAIRouteDropsUnsupportedMaxOutputTokens(t *testing.T) {
	handler, mockOpenAI, _ := setupOpenAIRouteTestDeps()

	body := []byte(`{
		"model": "claude-3-5-sonnet",
		"max_output_tokens": 64,
		"messages": [{"role": "user", "content": "hello"}]
	}`)

	rr, _ := doRawReq(t, handler, domain.ProviderAnthropic, body)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockOpenAI.chatCallCount)
	assert.Nil(t, mockOpenAI.lastMaxTokens)
	assert.Nil(t, mockOpenAI.lastMaxCompletion)
}
