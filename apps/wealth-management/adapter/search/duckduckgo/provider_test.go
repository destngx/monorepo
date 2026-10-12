package duckduckgo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const page = `<html><body>
<div class="result results_links result--ad"><a class="result__a" href="https://duckduckgo.com/y.js?ad=1">Ad</a></div>
<div class="result results_links web-result">
  <h2><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fwww.vnexpress.net%2Fvn-index&amp;rut=x">VN-Index <b>rallies</b></a></h2>
  <a class="result__snippet" href="#">Stocks rose   1.2% on <b>bank</b> gains.</a>
</div>
<div class="result results_links web-result">
  <h2><a class="result__a" href="https://cafef.vn/gold">Gold climbs</a></h2>
</div>
<div class="result results_links web-result"><a class="result__a" href="/relative">Broken</a></div>
</body></html>`

func TestGivenResultsPageWhenSearchThenParsesOrganicResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/html/" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := r.ParseForm(); err != nil || r.Form.Get("q") != "vn index" {
			t.Fatalf("expected the query in the form, got %v %v", r.Form, err)
		}
		_, _ = w.Write([]byte(page))
	}))
	defer server.Close()

	results, err := NewProvider(server.URL).Search(context.Background(), "vn index", 5)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected the ad and the relative link skipped, got %#v", results)
	}
	first := results[0]
	if first.URL != "https://www.vnexpress.net/vn-index" || first.Source != "vnexpress.net" || first.Title != "VN-Index rallies" || first.Snippet != "Stocks rose 1.2% on bank gains." {
		t.Fatalf("unexpected first result %#v", first)
	}
	if limited, _ := NewProvider(server.URL).Search(context.Background(), "vn index", 1); len(limited) != 1 {
		t.Fatalf("expected the limit applied, got %d", len(limited))
	}
}

func TestGivenChallengeStatusWhenSearchThenFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	if _, err := NewProvider(server.URL).Search(context.Background(), "x", 5); err == nil {
		t.Fatal("a non-200 answer must fail")
	}
}
