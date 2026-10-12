package google_sheets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"apps/wealth-management/domain"

	"google.golang.org/api/googleapi"
)

var resilienceConfig = domain.SheetsConfig{
	ClientID:      "client-id",
	ClientSecret:  "client-secret",
	RefreshToken:  "refresh-token",
	SpreadsheetID: "spreadsheet-123",
	RedirectURL:   "http://localhost",
}

// newResilienceServer answers the token endpoint and hands every other
// request to handler.
func newResilienceServer(t *testing.T, handler http.HandlerFunc) *SheetsClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "token", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	client := newTestClient(t, server.URL, resilienceConfig)
	client.retry.sleep = func(context.Context, time.Duration) error { return nil }
	return client
}

func writeAPIError(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]any{"code": code, "message": "fail"}})
}

func TestGivenSeveralRangesWhenReadingThenOneBatchGetRequestReturnsThemInOrder(t *testing.T) {
	var requests atomic.Int32
	client := newResilienceServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if !strings.HasSuffix(r.URL.Path, "/values:batchGet") {
			t.Errorf("path = %s, want values:batchGet", r.URL.Path)
		}
		if got := r.URL.Query()["ranges"]; len(got) != 2 || got[0] != "Accounts!A2:F" || got[1] != "Loans!A2:D" {
			t.Errorf("ranges = %v", got)
		}
		if r.URL.Query().Get("valueRenderOption") != "UNFORMATTED_VALUE" {
			t.Errorf("valueRenderOption = %q", r.URL.Query().Get("valueRenderOption"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"valueRanges": []any{
			map[string]any{"values": [][]any{{"Bank", 12.5, nil}}},
			map[string]any{},
		}})
	})

	rows, err := client.ReadSheets([]string{"Accounts!A2:F", "Loans!A2:D"})

	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || len(rows) != 2 || rows[0][0][1] != "12.5" || rows[0][0][2] != "" || len(rows[1]) != 0 {
		t.Fatalf("requests = %d rows = %#v", requests.Load(), rows)
	}
}

func TestGivenAThrottledReadWhenRetryingThenTheNextAttemptSucceeds(t *testing.T) {
	var requests atomic.Int32
	client := newResilienceServer(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			writeAPIError(w, http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": [][]any{{"ok"}}})
	})

	rows, err := client.ReadSheet("Accounts!A2:F")

	if err != nil || len(rows) != 1 || requests.Load() != 2 {
		t.Fatalf("rows = %v err = %v requests = %d", rows, err, requests.Load())
	}
}

func TestGivenAServerErrorThatPersistsWhenReadingThenItGivesUpAfterTheAttemptLimit(t *testing.T) {
	var requests atomic.Int32
	client := newResilienceServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		writeAPIError(w, http.StatusServiceUnavailable)
	})

	if _, err := client.ReadSheet("Accounts!A2:F"); err == nil {
		t.Fatal("want the server error")
	}
	if requests.Load() != 4 {
		t.Fatalf("requests = %d, want 4 attempts", requests.Load())
	}
}

func TestGivenABadRequestWhenReadingThenItIsNotRetried(t *testing.T) {
	var requests atomic.Int32
	client := newResilienceServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		writeAPIError(w, http.StatusBadRequest)
	})

	if _, err := client.ReadSheet("Accounts!A2:F"); err == nil {
		t.Fatal("want the error")
	}
	if requests.Load() != 1 {
		t.Fatalf("requests = %d, want 1", requests.Load())
	}
}

func TestGivenAWriteWhenTheServerFailsThenItIsNotRetriedButThrottlingIs(t *testing.T) {
	var requests atomic.Int32
	status := http.StatusServiceUnavailable
	client := newResilienceServer(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			writeAPIError(w, status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	})

	if err := client.UpdateRow("Accounts!A2", []any{"x"}); err == nil || requests.Load() != 1 {
		t.Fatalf("err = %v requests = %d, want a 503 write not retried", err, requests.Load())
	}

	requests.Store(0)
	status = http.StatusTooManyRequests
	if err := client.UpdateRow("Accounts!A2", []any{"x"}); err != nil || requests.Load() != 2 {
		t.Fatalf("err = %v requests = %d, want a 429 write retried", err, requests.Load())
	}
}

func TestGivenARetryAfterHeaderWhenComputingTheDelayThenItIsHonouredWithinTheCap(t *testing.T) {
	policy := defaultRetryPolicy()
	if got := policy.delay(1, nil); got != 500*time.Millisecond {
		t.Fatalf("first delay = %v", got)
	}
	if got := policy.delay(10, nil); got != 8*time.Second {
		t.Fatalf("capped delay = %v", got)
	}
	throttled := &googleapi.Error{Code: http.StatusTooManyRequests, Header: http.Header{"Retry-After": {"3"}}}
	if got := policy.delay(1, throttled); got != 3*time.Second {
		t.Fatalf("retry-after delay = %v, want 3s", got)
	}
	throttled.Header.Set("Retry-After", "60")
	if got := policy.delay(1, throttled); got != 8*time.Second {
		t.Fatalf("retry-after delay = %v, want the 8s cap", got)
	}
}
