package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGivenACodexAccountWhenReadingUsageThenBothWindowsAndResetTimesAreReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/usage" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get(providerHeader) != "openai" || r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("headers = %v", r.Header)
		}
		_, _ = w.Write([]byte(`{"provider":"openai","auth_mode":"oauth","source":"codex_endpoint",
			"rate_limits":{"primary":{"used_percent":12},"secondary":{"used_percent":40}},
			"display":{"5h_reset_at":"2026-10-22 15:04 UTC","weekly_reset_at":"bad"}}`))
	}))
	defer srv.Close()

	u, err := New(Config{BaseURL: srv.URL + "/v1/", APIKey: "k"}, nil).Usage(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if u.Provider != "openai" || u.AuthMode != "oauth" || len(u.Windows) != 2 || u.Totals != nil {
		t.Fatalf("usage = %+v", u)
	}
	five, week := u.Windows[0], u.Windows[1]
	want := time.Date(2026, 10, 22, 15, 4, 0, 0, time.UTC)
	if five.Name != "5h" || five.UsedPercent != 12 || five.ResetsAt == nil || !five.ResetsAt.Equal(want) {
		t.Fatalf("5h = %+v", five)
	}
	if week.Name != "weekly" || week.UsedPercent != 40 || week.ResetsAt != nil {
		t.Fatalf("weekly = %+v", week)
	}
}

func TestGivenAnAPIKeyAccountWhenReadingUsageThenTokenTotalsAreSummed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"object":"page","data":[
			{"start_time":1761000000,"results":[{"input_tokens":10,"output_tokens":2,"num_model_requests":1}]},
			{"start_time":1760900000,"results":[{"input_tokens":5,"output_tokens":1,"num_model_requests":2}]}]}`))
	}))
	defer srv.Close()

	u, err := New(Config{BaseURL: srv.URL}, nil).Usage(context.Background())

	if err != nil {
		t.Fatal(err)
	}
	if u.Provider != "openai" || u.AuthMode != "api_key" || len(u.Windows) != 0 || u.Totals == nil {
		t.Fatalf("usage = %+v", u)
	}
	tt := u.Totals
	if tt.InputTokens != 15 || tt.OutputTokens != 3 || tt.Requests != 3 || tt.Since == nil || tt.Since.Unix() != 1760900000 {
		t.Fatalf("totals = %+v", tt)
	}
}

func TestGivenAFailingGatewayWhenReadingUsageThenTheErrorCarriesNoProviderText(t *testing.T) {
	status, body := http.StatusUnauthorized, `Incorrect API key provided: sk-...abcd`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	r := New(Config{BaseURL: srv.URL}, nil)

	if _, err := r.Usage(context.Background()); err == nil || err.Error() != "the AI provider returned 401 Unauthorized" {
		t.Fatalf("status err = %v", err)
	}
	status, body = http.StatusOK, `{"something":"else"}`
	if _, err := r.Usage(context.Background()); err == nil || err.Error() != "the AI provider answered, but not as expected" {
		t.Fatalf("shape err = %v", err)
	}
	srv.Close()
	if _, err := r.Usage(context.Background()); err == nil || err.Error() != "the AI provider is unreachable at this address" {
		t.Fatalf("unreachable err = %v", err)
	}
}
