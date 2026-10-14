package openai

import (
	"apps/wealth-management/domain"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"testing"
)

func TestModelsListsSortedUniqueIDs(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("unexpected request %s %s %v", r.Method, r.URL.Path, r.Header)
		}
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"gpt-5"},{"id":" "},{"id":"claude-opus-5-5"},{"id":"gpt-5"}]}`)
	})

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"claude-opus-5-5", "gpt-5"}; !slices.Equal(models, want) {
		t.Fatalf("models %v, want %v", models, want)
	}
}

func TestModelsSendsTheConfiguredProvider(t *testing.T) {
	var provider string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		provider = r.Header.Get(providerHeader)
		_, _ = io.WriteString(w, `{"data":[]}`)
	})
	client.provider = "github-copilot"

	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if provider != "github-copilot" || models == nil || len(models) != 0 {
		t.Fatalf("provider %q, models %#v", provider, models)
	}
}

func TestModelsFailuresAreUnavailable(t *testing.T) {
	for name, handler := range map[string]http.HandlerFunc{
		"status": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "nope", http.StatusBadGateway) },
		"body":   func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, `not json`) },
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := newTestClient(t, handler).Models(context.Background()); !errors.Is(err, domain.ErrAIUnavailable) {
				t.Fatalf("want ErrAIUnavailable, got %v", err)
			}
		})
	}
}
