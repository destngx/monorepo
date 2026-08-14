package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/internal/transport/http/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicCompatibilityHandlerGet(t *testing.T) {
	handler := NewAnthropicCompatibilityHandler()
	req := httptest.NewRequest(http.MethodGet, "/api/hello", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, common.ContentTypeJSON, rr.Header().Get(common.HeaderContentType))
	assert.Equal(t, AnthropicHelloResponse, rr.Body.String())
}

func TestAnthropicCompatibilityHandlerHead(t *testing.T) {
	handler := NewAnthropicCompatibilityHandler()
	req := httptest.NewRequest(http.MethodHead, "/api/hello", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, common.ContentTypeJSON, rr.Header().Get(common.HeaderContentType))
	assert.Equal(t, AnthropicHelloResponseLength, rr.Header().Get(common.HeaderContentLength))
	assert.Empty(t, rr.Body.String())
}

func TestAnthropicCompatibilityHandlerRejectsUnsupportedMethods(t *testing.T) {
	handler := NewAnthropicCompatibilityHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/hello", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}
