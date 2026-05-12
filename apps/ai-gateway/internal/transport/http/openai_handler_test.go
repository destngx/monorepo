package httptransport

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"

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
