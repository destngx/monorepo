package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestChat_UsesResponsesEndpointForGPT54Mini(t *testing.T) {
	var requestedPath string
	var requestedBody map[string]any
	p := New("gh-token", "individual", 0)
	p.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		requestedPath = r.URL.Path
		require.Equal(t, "Bearer copilot-token", r.Header.Get(headerAuthorization))
		require.Equal(t, defaultIntegrationID, r.Header.Get(headerIntegrationID))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&requestedBody))

		body := strings.Join([]string{
			`data: {"type":"response.created","response":{"id":"resp_123","created_at":123,"model":"gpt-5.4-mini"}}`,
			`data: {"type":"response.output_text.delta","delta":"hello"}`,
			`data: {"type":"response.completed","response":{"id":"resp_123","created_at":123,"model":"gpt-5.4-mini","usage":{"input_tokens":3,"output_tokens":2,"total_tokens":5}}}`,
			"data: [DONE]",
			"",
		}, "\n\n")

		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{headerContentType: []string{contentTypeJSON}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	})}
	p.cachedToken = "copilot-token"
	p.expiresAt = time.Now().Add(time.Hour).Unix()
	p.copilotAPIBase = "https://copilot.example.test"

	resp, err := p.Chat(t.Context(), domain.ChatRequest{
		Model:    domain.ModelGPT54Mini,
		Messages: []domain.Message{{Role: domain.RoleUser, Content: "hello"}},
	})
	require.NoError(t, err)

	assert.Equal(t, pathResponses, requestedPath)
	assert.Equal(t, domain.ModelGPT54Mini, requestedBody["model"])
	assert.Equal(t, true, requestedBody["stream"])
	assert.Equal(t, "hello", resp.Choices[0].Message.Content)
	assert.Equal(t, domain.Usage{PromptTokens: 3, CompletionTokens: 2, TotalTokens: 5}, resp.Usage)
}

func TestNewChatRequest_SetsCopilotClientHeaders(t *testing.T) {
	p := New("gh-token", "individual", 0, ClientHeaders{
		EditorVersion:       "vscode/1.99.0",
		EditorPluginVersion: "copilot-chat/9.9.9",
		IntegrationID:       "test-integration",
		UserAgent:           "test-agent/1.0",
	})
	p.cachedToken = "copilot-token"
	p.expiresAt = time.Now().Add(time.Hour).Unix()
	p.copilotAPIBase = "https://copilot.example.test"

	req, err := p.newChatRequest(t.Context(), domain.ChatRequest{
		Model:    domain.ModelGPT5Mini,
		Messages: []domain.Message{{Role: domain.RoleUser, Content: "hello"}},
	}, true)
	require.NoError(t, err)

	assert.Equal(t, "Bearer copilot-token", req.Header.Get(headerAuthorization))
	assert.Equal(t, "vscode/1.99.0", req.Header.Get(headerEditorVersion))
	assert.Equal(t, "copilot-chat/9.9.9", req.Header.Get(headerEditorPluginVer))
	assert.Equal(t, "test-integration", req.Header.Get(headerIntegrationID))
	assert.Equal(t, "test-agent/1.0", req.Header.Get(headerUserAgent))
}

func TestBatchMessagesWithOversizedToolCalls(t *testing.T) {
	p := &Provider{verbose: 1}

	t.Run("UnderLimit", func(t *testing.T) {
		msgs := []domain.Message{
			{Role: domain.RoleUser, Content: "hi"},
			{Role: domain.RoleAssistant, ToolCalls: []domain.ToolCall{{ID: "1"}}},
			{Role: domain.RoleTool, ToolCallID: "1", Content: "res"},
		}
		result := p.batchMessagesWithOversizedToolCalls(msgs)
		assert.Equal(t, msgs, result)
	})

	t.Run("OverLimitWithResponses", func(t *testing.T) {
		// Mock max limit to 2 for easier testing
		// We'll temporarily override the constant if possible, but it's a const.
		// Since it's a constant, we'll have to use the real limit or
		// we can make it a variable in the provider for testing.
		// For now, let's assume we use a large number but for test we'll use a small one if we can.
		// Wait, maxToolCallsPerMessage is a constant 128.

		toolCalls := make([]domain.ToolCall, 130)
		toolMsgs := make([]domain.Message, 130)
		for i := 0; i < 130; i++ {
			id := fmt.Sprintf("call_%d", i)
			toolCalls[i] = domain.ToolCall{ID: id, Type: "function"}
			toolMsgs[i] = domain.Message{Role: domain.RoleTool, ToolCallID: id, Content: fmt.Sprintf("res_%d", i)}
		}

		input := append([]domain.Message{
			{Role: domain.RoleUser, Content: "complex task"},
			{Role: domain.RoleAssistant, Content: "thinking", ToolCalls: toolCalls},
		}, toolMsgs...)

		result := p.batchMessagesWithOversizedToolCalls(input)

		// Expected:
		// 1. User
		// 2. Assistant (1-128)
		// 3. Tool (1-128)
		// 4. Assistant (129-130)
		// 5. Tool (129-130)

		assert.Greater(t, len(result), len(input))

		// Check interleaving
		assert.Equal(t, domain.RoleAssistant, result[1].Role)
		assert.Equal(t, 128, len(result[1].ToolCalls))
		assert.Equal(t, "thinking", result[1].Content)

		assert.Equal(t, domain.RoleTool, result[2].Role)
		assert.Equal(t, "call_0", result[2].ToolCallID)

		assert.Equal(t, domain.RoleTool, result[129].Role)
		assert.Equal(t, "call_127", result[129].ToolCallID)

		assert.Equal(t, domain.RoleAssistant, result[130].Role)
		assert.Equal(t, 2, len(result[130].ToolCalls))
		assert.Equal(t, "", result[130].Content) // Content should be empty in subsequent batches

		assert.Equal(t, domain.RoleTool, result[131].Role)
		assert.Equal(t, "call_128", result[131].ToolCallID)
	})

	t.Run("OverLimitAtEnd", func(t *testing.T) {
		toolCalls := make([]domain.ToolCall, 130)
		for i := 0; i < 130; i++ {
			toolCalls[i] = domain.ToolCall{ID: fmt.Sprintf("call_%d", i)}
		}
		input := []domain.Message{
			{Role: domain.RoleAssistant, ToolCalls: toolCalls},
		}
		result := p.batchMessagesWithOversizedToolCalls(input)

		// Should still split even if no responses
		assert.Equal(t, 2, len(result))
		assert.Equal(t, 128, len(result[0].ToolCalls))
		assert.Equal(t, 2, len(result[1].ToolCalls))
	})
}
