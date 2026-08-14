package openai

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"
	"apps/ai-gateway/internal/transport/http/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenAIHandler_KeepsGPT54MiniCopilotRequestOnGitHubCopilot(t *testing.T) {
	reg := service.NewRegistry(&config.Config{})
	reg.Mapper = service.NewModelMapper(domain.ProviderGitHubCopilot)

	mockCopilot := &MockTestProvider{name: domain.ProviderGitHubCopilot}
	mockOpenAI := &MockTestProvider{name: domain.ProviderOpenAI}
	reg.RegisterForTest(mockCopilot)
	reg.RegisterForTest(mockOpenAI)

	body, err := json.Marshal(domain.ChatRequest{
		Model:    domain.ModelGPT54Mini,
		Messages: []domain.Message{{Role: domain.RoleUser, Content: "hello"}},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, pathChatCompletions, bytes.NewReader(body))
	req.Header.Set(domain.HeaderAIProvider, domain.ProviderGitHubCopilot)
	rr := httptest.NewRecorder()

	NewOpenAIHandler(reg).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockCopilot.chatCallCount)
	assert.Equal(t, 0, mockOpenAI.chatCallCount)
	assert.Equal(t, domain.ModelGPT54Mini, mockCopilot.lastChatModel)
}

func TestResponsesHandler_RoutesNativeResponsesRequest(t *testing.T) {
	reg := service.NewRegistry(&config.Config{})
	reg.Mapper = service.NewModelMapper(domain.ProviderGitHubCopilot)

	mockCopilot := &MockTestProvider{name: domain.ProviderGitHubCopilot}
	reg.RegisterForTest(mockCopilot)

	body := []byte(`{
		"model": "gpt-5.4-mini",
		"instructions": "You are concise.",
		"input": [{"role": "user", "content": [{"type": "input_text", "text": "hello"}]}]
	}`)

	req := httptest.NewRequest(http.MethodPost, pathResponses, bytes.NewReader(body))
	req.Header.Set(domain.HeaderAIProvider, domain.ProviderGitHubCopilot)
	rr := httptest.NewRecorder()

	NewResponsesHandler(reg).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 1, mockCopilot.responsesCallCount)
	assert.Equal(t, domain.ModelGPT54Mini, mockCopilot.lastResponsesModel)
}

func TestResponsesHandler_EmitsTypedFailureEventAfterStreamStarts(t *testing.T) {
	reg := service.NewRegistry(&config.Config{})
	reg.Mapper = service.NewModelMapper(domain.ProviderOpenAI)

	mockOpenAI := &MockTestProvider{
		name:               domain.ProviderOpenAI,
		responsesStreamErr: errors.New("upstream rejected parameter"),
	}
	reg.RegisterForTest(mockOpenAI)

	body := []byte(`{"model":"gpt-5.6-luna","input":[],"stream":true}`)
	req := httptest.NewRequest(http.MethodPost, pathResponses, bytes.NewReader(body))
	req.Header.Set(domain.HeaderAIProvider, domain.ProviderOpenAI)
	rr := httptest.NewRecorder()

	NewResponsesHandler(reg).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	scanner := bufio.NewScanner(rr.Body)
	require.True(t, scanner.Scan())
	line := scanner.Text()
	require.Contains(t, line, common.SSEDataPrefix)
	var event map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, common.SSEDataPrefix)), &event))
	assert.Equal(t, responsesEventFailed, event["type"])
	assert.Contains(t, event["error"], "upstream rejected parameter")
}
