package tavily

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGivenKeyWhenSearchThenSendsBearerAndMapsResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.URL.Path != "/search" || r.Header.Get("Authorization") != "Bearer tvly-key" || body["query"] != "gold" || body["max_results"] != 3.0 {
			t.Fatalf("unexpected request %s %v %v", r.URL.Path, r.Header, body)
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"Gold","url":"https://www.reuters.com/gold","content":"Gold rose."},{"title":"bad","url":"","content":"x"}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(server.URL, " tvly-key ")
	if err != nil {
		t.Fatal(err)
	}
	results, err := provider.Search(context.Background(), "gold", 3)
	if err != nil || len(results) != 1 || results[0].Source != "reuters.com" || results[0].Snippet != "Gold rose." {
		t.Fatalf("unexpected results %#v %v", results, err)
	}
	if _, err := NewProvider(server.URL, ""); err == nil {
		t.Fatal("a missing key must be rejected")
	}
}
