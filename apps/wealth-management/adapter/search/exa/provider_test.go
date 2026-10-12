package exa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGivenKeyWhenSearchThenSendsHeaderAndFallsBackToSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Header.Get("x-api-key") != "exa-key" || body["numResults"] != 5.0 {
			t.Fatalf("unexpected request %v %v", r.Header, body)
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"Fed","url":"https://ft.com/fed","summary":"Rates held."},{"title":"SBV","url":"https://sbv.gov.vn/a","highlights":["Policy steady."]}]}`))
	}))
	defer server.Close()

	provider, err := NewProvider(server.URL, "exa-key")
	if err != nil {
		t.Fatal(err)
	}
	results, err := provider.Search(context.Background(), "fed", 5)
	if err != nil || len(results) != 2 || results[0].Snippet != "Rates held." || results[1].Snippet != "Policy steady." {
		t.Fatalf("unexpected results %#v %v", results, err)
	}
}
