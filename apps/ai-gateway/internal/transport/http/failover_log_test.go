package httptransport

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerReportsFallbackProviderAndFailoverDetails(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = SetLogProvider(r, domain.ProviderOpenAI)
		meta := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta)
		meta.Failover = &domain.FailoverLogMeta{
			Primary:       domain.ProviderOpenAI,
			Fallback:      domain.ProviderGitHubCopilot,
			Reason:        domain.FailoverReasonPrimaryLimit,
			TriggerModel:  "gpt-5",
			TriggerError:  "429 usage_limit_reached",
			FallbackError: "copilot unavailable",
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/responses", nil))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "Request", entry["msg"])
	assert.Equal(t, domain.ProviderGitHubCopilot, entry["provider"])
	assert.Equal(t, domain.FailoverReasonPrimaryLimit, entry["failover"])
	assert.Equal(t, domain.ProviderOpenAI, entry["failover_primary"])
	assert.Equal(t, domain.ProviderGitHubCopilot, entry["failover_fallback"])
	assert.Equal(t, "gpt-5", entry["failover_trigger_model"])
	assert.Equal(t, "429 usage_limit_reached", entry["failover_trigger_error"])
	assert.Equal(t, "copilot unavailable", entry["failover_fallback_error"])
}

func TestLoggerKeepsPrimaryWhenStreamStartedBeforeLimit(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = SetLogProvider(r, domain.ProviderOpenAI)
		meta := r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta)
		meta.Failover = &domain.FailoverLogMeta{
			Primary:  domain.ProviderOpenAI,
			Fallback: domain.ProviderGitHubCopilot,
			Reason:   domain.FailoverReasonStreamStarted,
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/responses", nil))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, domain.ProviderOpenAI, entry["provider"])
	assert.Equal(t, domain.FailoverReasonStreamStarted, entry["failover"])
	assert.NotContains(t, entry, "failover_fallback_error")
}
