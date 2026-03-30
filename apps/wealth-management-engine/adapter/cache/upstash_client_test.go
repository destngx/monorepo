package cache

import (
	"apps/wealth-management-engine/domain"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGivenSetRequestWhenCacheAcceptsThenValueIsWritten(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/set/user:1/balance" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.RawQuery != "EX=300" {
			t.Fatalf("expected EX=300, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"OK"}`))
	}))
	defer server.Close()

	client, err := NewClient(domain.CacheConfig{
		RESTURL:   server.URL,
		RESTToken: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to build cache client: %v", err)
	}

	err = client.Set(context.Background(), "user:1", "balance", 300)
	if err != nil {
		t.Fatalf("expected set to pass, got error: %v", err)
	}
}

func TestGivenCachedValueWhenGetThenReturnsValueAndFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/get/user:1" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result":"42"}`))
	}))
	defer server.Close()

	client, err := NewClient(domain.CacheConfig{
		RESTURL:   server.URL,
		RESTToken: "test-token",
	})
	if err != nil {
		t.Fatalf("failed to build cache client: %v", err)
	}

	value, found, err := client.Get(context.Background(), "user:1")
	if err != nil {
		t.Fatalf("expected get to pass, got error: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	if value != "42" {
		t.Fatalf("expected value 42, got %s", value)
	}
}
