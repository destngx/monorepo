package httptransport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// serveWithFailover runs a request through the Logger and Metrics middleware
// (in the same order as main.go) with a handler that reports the given failover.
func serveWithFailover(t *testing.T, failover *domain.FailoverLogMeta) domain.MetricsSummary {
	t.Helper()
	collector := service.NewMetricsCollector(10, "", 0)
	handler := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := r.Context().Value(domain.MetricsPayloadKey).(*domain.MetricsPayload)
		payload.Provider = domain.ProviderOpenAI
		payload.Model = "gpt-5"
		r.Context().Value(domain.LogMetaKey).(*domain.RequestLogMeta).Failover = failover
	}), Logger, Metrics(collector))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
	return collector.Summary()
}

func TestMetricsRecordsFallbackProviderWhenFailoverServedRequest(t *testing.T) {
	summary := serveWithFailover(t, &domain.FailoverLogMeta{
		Primary:      domain.ProviderOpenAI,
		Fallback:     domain.ProviderGitHubCopilot,
		Reason:       domain.FailoverReasonPrimaryLimit,
		TriggerError: "429 usage_limit_reached",
	})

	require.Len(t, summary.RecentRequests, 1)
	recent := summary.RecentRequests[0]
	assert.Equal(t, domain.ProviderGitHubCopilot, recent.Provider)
	assert.Equal(t, &domain.FailoverRecord{
		Primary:  domain.ProviderOpenAI,
		Fallback: domain.ProviderGitHubCopilot,
		Reason:   domain.FailoverReasonPrimaryLimit,
	}, recent.Failover)
	assert.Contains(t, summary.ByProvider, domain.ProviderGitHubCopilot)
	assert.NotContains(t, summary.ByProvider, domain.ProviderOpenAI)
}

func TestMetricsKeepsPrimaryWhenStreamStartedBeforeLimit(t *testing.T) {
	summary := serveWithFailover(t, &domain.FailoverLogMeta{
		Primary:  domain.ProviderOpenAI,
		Fallback: domain.ProviderGitHubCopilot,
		Reason:   domain.FailoverReasonStreamStarted,
	})

	require.Len(t, summary.RecentRequests, 1)
	recent := summary.RecentRequests[0]
	assert.Equal(t, domain.ProviderOpenAI, recent.Provider)
	require.NotNil(t, recent.Failover)
	assert.Equal(t, domain.FailoverReasonStreamStarted, recent.Failover.Reason)
}

func TestMetricsOmitsFailoverWhenPrimaryServedRequest(t *testing.T) {
	summary := serveWithFailover(t, nil)

	require.Len(t, summary.RecentRequests, 1)
	assert.Equal(t, domain.ProviderOpenAI, summary.RecentRequests[0].Provider)
	assert.Nil(t, summary.RecentRequests[0].Failover)
}
