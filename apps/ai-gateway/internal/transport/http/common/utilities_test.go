package common

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"apps/ai-gateway/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestSetMetricsAttributesFallbackProvider(t *testing.T) {
	meta := &domain.RequestLogMeta{
		Provider: domain.ProviderOpenAI,
		Failover: &domain.FailoverLogMeta{
			Primary:  domain.ProviderOpenAI,
			Fallback: domain.ProviderGitHubCopilot,
			Reason:   domain.FailoverReasonPrimaryCooldown,
		},
	}
	payload := &domain.MetricsPayload{}
	ctx := context.WithValue(context.Background(), domain.LogMetaKey, meta)
	ctx = context.WithValue(ctx, domain.MetricsPayloadKey, payload)
	r := httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)

	SetMetrics(r, domain.ProviderOpenAI, "gpt-5", "gpt-5", domain.Usage{}, false, nil)

	assert.Equal(t, domain.ProviderGitHubCopilot, payload.Provider)
}

func TestSetMetricsKeepsProviderWithoutFailover(t *testing.T) {
	payload := &domain.MetricsPayload{}
	ctx := context.WithValue(context.Background(), domain.LogMetaKey, &domain.RequestLogMeta{Provider: domain.ProviderOpenAI})
	ctx = context.WithValue(ctx, domain.MetricsPayloadKey, payload)
	r := httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)

	SetMetrics(r, domain.ProviderOpenAI, "gpt-5", "gpt-5", domain.Usage{}, false, nil)

	assert.Equal(t, domain.ProviderOpenAI, payload.Provider)
}
