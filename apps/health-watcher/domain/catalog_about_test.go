package domain

import (
	"strings"
	"testing"
)

func TestGivenTheAboutDataWhenCheckedThenEveryEntryIsACatalogMetricWithASourcedQuote(t *testing.T) {
	if len(metricAbout) == 0 {
		t.Fatal("no metric has a description")
	}
	for code := range metricAbout {
		if _, ok := DefaultCatalog.Metric(code); !ok {
			t.Errorf("about entry %q is not a catalog code", code)
			continue
		}
		a, ok := AboutMetric(code)
		if !ok {
			t.Errorf("AboutMetric(%q) not found", code)
			continue
		}
		if strings.TrimSpace(a.Quote) == "" {
			t.Errorf("%s: empty quote", code)
		}
		if a.Title == "" {
			t.Errorf("%s: empty page title", code)
		}
		site := map[string]string{AboutSource: "https://medlineplus.gov/",
			AboutSourceNCI: "https://www.cancer.gov/publications/dictionaries/cancer-terms/def/"}[a.Source]
		if site == "" || !strings.HasPrefix(a.URL, site) || strings.Contains(a.URL, "?") {
			t.Errorf("%s: URL %q from %q is not a plain page of a known source", code, a.URL, a.Source)
		}
		if a.Fetched != AboutFetched || a.Fetched == "" {
			t.Errorf("%s: source %q fetched %q", code, a.Source, a.Fetched)
		}
	}
}

func TestGivenACodeWhenAskingAboutItThenOnlyDescribedMetricsAnswer(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"glucose", true},
		{"wbc", true},
		{"neutrophils_pct", true},
		{"urine_nitrite", true},
		{"granulocytes_abs", true},
		{"mid_abs", false},
		{"height", false},
		{"no_such_metric", false},
	}
	for _, tc := range cases {
		a, ok := AboutMetric(tc.code)
		if ok != tc.want {
			t.Errorf("AboutMetric(%q) ok = %v; want %v", tc.code, ok, tc.want)
		}
		if !ok && a != (MetricAbout{}) {
			t.Errorf("AboutMetric(%q) = %+v; want zero value", tc.code, a)
		}
	}
}
