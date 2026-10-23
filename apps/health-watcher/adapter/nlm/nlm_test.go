package nlm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"apps/health-watcher/domain"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.Client())
	c.tablesURL, c.connectURL = srv.URL+"/tables", srv.URL+"/connect"
	c.now = func() time.Time { return time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC) }
	return c
}

func TestGivenALOINCCodeWhenNamingItThenOnlyAnExactCodeMatches(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("terms") != "26499-4" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[2,["26499-4","26499-40"],null,[["26499-40","Other"],["26499-4","Neutrophils [#/volume] in Blood"]]]`))
	})
	name, ok, err := c.Name(context.Background(), "26499-4")
	if err != nil || !ok || name != "Neutrophils [#/volume] in Blood" {
		t.Fatalf("name = %q %v %v", name, ok, err)
	}
	c = testClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`[0,[],null,[]]`)) })
	if _, ok, err := c.Name(context.Background(), "9999-9"); ok || err != nil {
		t.Fatalf("unknown code: ok=%v err=%v", ok, err)
	}
}

func TestGivenAMedlinePlusPageWhenDescribingThenTheOpeningSentencesAreQuotedWordForWord(t *testing.T) {
	long := strings.Repeat("Word ", 60) + "end. " + strings.Repeat("More ", 60) + "end."
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"feed":{"entry":[{"title":{"_value":"Ferritin Blood Test"},"link":[{"href":"https://medlineplus.gov/lab-tests/ferritin-blood-test/?utm_source=x"}],
			"summary":{"_value":"<h3>What is it?</h3><p>A ferritin test measures <b>ferritin</b> &amp; iron. ` + long + `</p><p>Second.</p>"}}]}}`))
	})
	a, ok, err := c.About(context.Background(), "2276-4")
	if err != nil || !ok {
		t.Fatalf("about: ok=%v err=%v", ok, err)
	}
	want := domain.MetricAbout{Quote: "A ferritin test measures ferritin & iron. " + strings.TrimSpace(strings.Repeat("Word ", 60)) + " end.",
		Source: domain.AboutSource, Title: "Ferritin Blood Test", URL: "https://medlineplus.gov/lab-tests/ferritin-blood-test/", Fetched: "2026-10-23"}
	if a != want {
		t.Fatalf("about = %+v", a)
	}
}

func TestGivenNoMedlinePlusPageWhenDescribingThenThereIsNone(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"feed":{"entry":[]}}`)) })
	if _, ok, err := c.About(context.Background(), "1-8"); ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}
