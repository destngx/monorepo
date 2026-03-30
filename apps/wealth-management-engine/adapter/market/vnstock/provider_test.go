package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGivenVNStockHealthEndpointWhenHealthThenReturnsProviderStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	provider, err := NewProvider(domain.MarketDataProviderConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("new provider error: %v", err)
	}

	health, err := provider.Health(context.Background())
	if err != nil {
		t.Fatalf("health error: %v", err)
	}
	if health.Provider != "vnstock" || health.Status != "ok" {
		t.Fatalf("unexpected health response: %#v", health)
	}
}
