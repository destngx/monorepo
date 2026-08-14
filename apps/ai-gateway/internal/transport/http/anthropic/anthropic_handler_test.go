package anthropic

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

func TestIsWebSearchToolMatchesVersionedNames(t *testing.T) {
	for _, value := range []string{"web_search", "web_search_20241022", "web_search_20250305", "WEB_SEARCH_PREVIEW"} {
		if !isWebSearchTool(value) {
			t.Errorf("expected %q to be recognized as web search", value)
		}
	}
	for _, value := range []string{"search", "web_searchx", "code_execution"} {
		if isWebSearchTool(value) {
			t.Errorf("did not expect %q to be recognized as web search", value)
		}
	}
}

func TestConvertAnthropicWebSearchToolChoiceFallsBackToAuto(t *testing.T) {
	req := convertFromAnthropicRequest(anthropic.Request{
		Model: "claude-haiku-4-5-20251001",
		ToolChoice: map[string]any{
			"type": "tool",
			"name": "web_search_20250305",
		},
	}, domain.ProviderOpenAI)

	assert.Equal(t, "auto", req.ToolChoice)
}

func TestConvertAnthropicWebSearchToolPreservesHostedTool(t *testing.T) {
	req := convertFromAnthropicRequest(anthropic.Request{
		Tools: []anthropic.Tool{{Name: "web_search", Type: "web_search_20250305"}},
	}, domain.ProviderOpenAI)

	if len(req.Tools) != 1 || req.Tools[0].Type != domain.ToolTypeWebSearch {
		t.Fatalf("expected hosted web-search tool, got %#v", req.Tools)
	}
}

type MockTestProvider struct {
	name                string
	chatCallCount       int
	streamCallCount     int
	responsesCallCount  int
	lastResponsesModel  string
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
func (m *MockTestProvider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	m.responsesCallCount++
	m.lastResponsesModel = req.Model
	return &domain.ResponsesResponse{
		"id":    "resp_mock",
		"model": req.Model,
		"usage": map[string]any{
			"input_tokens":  1,
			"output_tokens": 2,
			"total_tokens":  3,
		},
	}, nil
}
func (m *MockTestProvider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	m.responsesCallCount++
	m.lastResponsesModel = req.Model
	return domain.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}, nil
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
	reg.Mapper = service.NewModelMapper(domain.ProviderGitHubCopilot)
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
	reg.Mapper = service.NewModelMapper(domain.ProviderGitHubCopilot)

	mockGitHubCopilot := &MockTestProvider{name: domain.ProviderGitHubCopilot}
	reg.RegisterForTest(mockGitHubCopilot)

	return NewAnthropicHandler(reg), mockGitHubCopilot
}

func setupOpenAIRouteTestDeps() (*AnthropicHandler, *MockTestProvider, *MockTestProvider) {
	cfg := &config.Config{AnthropicRoute: "openai-gpt-5.4-mini-low"}
	reg := service.NewRegistry(cfg)
	reg.Mapper = service.NewModelMapper(domain.ProviderGitHubCopilot)

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

func TestAnthropicHandler_AllowsVersionedWebSearchTool_NonAnthropic(t *testing.T) {
	handler, mockOpenAI, _ := setupTestDeps()

	req := anthropic.Request{
		Tools: []anthropic.Tool{
			{Name: "web_search", Type: "web_search_20250305"},
			{Name: "valid_tool", Type: "function"},
		},
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockOpenAI.chatCallCount)
}

func TestAnthropicHandler_AllowsVersionedWebSearchMessageBlock_NonAnthropic(t *testing.T) {
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

	rr, _ := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockOpenAI.chatCallCount)
}

func TestAnthropicHandler_AllowsVersionedWebSearchToolChoice_NonAnthropic(t *testing.T) {
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

	rr, _ := doReq(t, handler, domain.ProviderOpenAI, "gpt-4o", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockOpenAI.chatCallCount)
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

func TestAnthropicHandler_AllowsStreamVersionedWebSearch_NonAnthropic(t *testing.T) {
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

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
	assert.Equal(t, 1, mockOpenAI.streamCallCount)
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

func TestAnthropicHandler_DefaultRouteInterceptorUsesGitHubCopilotDefaultHigh(t *testing.T) {
	handler, mockGitHubCopilot := setupDefaultRouteTestDeps()

	req := anthropic.Request{
		Messages: []anthropic.Message{{Role: "user", Content: "hello"}},
	}

	rr, _ := doReq(t, handler, "", "", req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockGitHubCopilot.chatCallCount)
	assert.Equal(t, domain.ModelDefault, mockGitHubCopilot.lastChatModel)
	assert.Equal(t, domain.ReasoningEffortLow, mockGitHubCopilot.lastReasoningEffort)
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

func TestNormalizeStreamToolArgumentsRemovesEmptyReadPages(t *testing.T) {
	got, changed := normalizeStreamToolArguments(`{"command":"ls","pages":""}`)
	assert.True(t, changed)
	assert.JSONEq(t, `{"command":"ls"}`, got)
}

func TestNormalizeStreamToolArgumentsPreservesValidPages(t *testing.T) {
	input := `{"command":"ls","pages":"1"}`
	got, changed := normalizeStreamToolArguments(input)
	assert.False(t, changed)
	assert.Equal(t, input, got)
}

func TestNormalizeStreamToolArgumentsPreservesPartialJSON(t *testing.T) {
	input := `{"command":"ls","pages":"`
	got, changed := normalizeStreamToolArguments(input)
	assert.False(t, changed)
	assert.Equal(t, input, got)
}

func TestConvertToAnthropicStreamPreservesArgumentsFromFirstToolChunk(t *testing.T) {
	input := "data: {\"id\":\"chat-1\",\"model\":\"gpt-5.4-mini\",\"choices\":[{" +
		"\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call-1\",\"type\":\"function\",\"function\":{\"name\":\"Bash\",\"arguments\":\"{\\\"command\\\":\\\"git status --short\\\"}\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n"

	var output bytes.Buffer
	count, err := convertToAnthropicStream(bytes.NewBufferString(input), &output, "claude-haiku-4-5-20251001")
	assert.NoError(t, err)
	assert.Positive(t, count)
	assert.Contains(t, output.String(), `"model":"claude-haiku-4-5-20251001"`)
	assert.NotContains(t, output.String(), `"model":"gpt-5.4-mini"`)
	assert.Contains(t, output.String(), "git status --short")
	assert.NotContains(t, output.String(), "partial_json\\\":\\\"{}")
}

func TestConvertToAnthropicStreamSuppressesEmptyToolCall(t *testing.T) {
	input := "data: {\"id\":\"chat-1\",\"model\":\"gpt-5.4-mini\",\"choices\":[{" +
		"\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"empty\",\"type\":\"function\",\"function\":{\"name\":\"Read\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"valid\",\"type\":\"function\",\"function\":{\"name\":\"Read\",\"arguments\":\"{\\\"file_path\\\":\\\"README.md\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n"

	var output bytes.Buffer
	_, err := convertToAnthropicStream(bytes.NewBufferString(input), &output, "claude-haiku-4-5-20251001")
	assert.NoError(t, err)
	assert.NotContains(t, output.String(), `"id":"empty"`)
	assert.Contains(t, output.String(), `"id":"valid"`)
	assert.Contains(t, output.String(), "README.md")
}

func TestConvertToAnthropicStreamCompletesParallelToolCallsBeforeStopping(t *testing.T) {
	input := "data: {\"id\":\"chat-1\",\"model\":\"gpt-5.6\",\"choices\":[{" +
		"\"delta\":{\"tool_calls\":[{\"id\":\"call-0\",\"index\":0,\"function\":{\"name\":\"Bash\",\"arguments\":\"{\\\"command\\\":\\\"ls\\\"}\"}},{\"id\":\"call-1\",\"index\":1,\"function\":{\"name\":\"Bash\",\"arguments\":\"{\\\"command\\\":\\\"pwd\\\"}\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n"

	var output bytes.Buffer
	_, err := convertToAnthropicStream(bytes.NewBufferString(input), &output, "claude-haiku-4-5-20251001")
	assert.NoError(t, err)
	result := output.String()
	assert.Contains(t, result, `"partial_json":"{\"command\":\"ls\"}"`)
	assert.Contains(t, result, `"partial_json":"{\"command\":\"pwd\"}"`)
}

func TestConvertToAnthropicStreamPreservesUsageAndCacheCounters(t *testing.T) {
	input := "data: {\"id\":\"chat-1\",\"model\":\"gpt-5.6-luna\",\"choices\":[{\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":101,\"completion_tokens\":7,\"total_tokens\":108,\"prompt_tokens_details\":{\"cached_tokens\":80,\"cache_write_tokens\":12}}}\n\n" +
		"data: [DONE]\n\n"

	var output bytes.Buffer
	_, err := convertToAnthropicStream(bytes.NewBufferString(input), &output, "claude-haiku-4-5-20251001")
	assert.NoError(t, err)
	assert.Contains(t, output.String(), `"input_tokens":101`)
	assert.Contains(t, output.String(), `"output_tokens":7`)
	assert.Contains(t, output.String(), `"cache_read_input_tokens":80`)
	assert.Contains(t, output.String(), `"cache_creation_input_tokens":12`)
}

func TestConvertToAnthropicStreamReportsMissingWebSearchSourcesAsUnavailable(t *testing.T) {
	input := "data: {\"object\":\"chat.completion.chunk\",\"web_search\":{\"id\":\"ws_1\",\"query\":\"current Node.js LTS\"},\"choices\":[]}\n\n" +
		"data: {\"object\":\"chat.completion.chunk\",\"web_search_result\":{\"id\":\"ws_1\",\"error\":\"unavailable\"},\"choices\":[]}\n\n" +
		"data: [DONE]\n\n"

	var output bytes.Buffer
	_, err := convertToAnthropicStream(bytes.NewBufferString(input), &output, "claude-haiku-4-5-20251001")
	assert.NoError(t, err)
	assert.Contains(t, output.String(), `"type":"web_search_tool_result_error"`)
	assert.Contains(t, output.String(), `"error_code":"unavailable"`)
	assert.NotContains(t, output.String(), "No search results were returned")
	messageStart := bytes.Index(output.Bytes(), []byte("event: message_start"))
	contentStart := bytes.Index(output.Bytes(), []byte("event: content_block_start"))
	assert.GreaterOrEqual(t, messageStart, 0)
	assert.Greater(t, contentStart, messageStart)
	assert.Contains(t, output.String(), `"id":"msg_ai_gateway"`)
}
