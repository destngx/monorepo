package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderNameFromModelsPath(t *testing.T) {
	assert.Equal(t, "", providerNameFromModelsPath("/v1/models"))
	assert.Equal(t, domain.ProviderOpenAI, providerNameFromModelsPath("/v1/models/openai"))
	assert.Equal(t, "", providerNameFromModelsPath("/v1/models/openai/extra"))
}

func TestModelsHandlerSupportsExactModelsPath(t *testing.T) {
	reg := service.NewRegistry(&config.Config{})
	provider := &MockTestProvider{name: domain.ProviderOpenAI}
	reg.RegisterForTest(provider)

	handler := NewModelsHandler(reg)
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set(domain.HeaderAIProvider, domain.ProviderOpenAI)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `"object":"list"`)
}
