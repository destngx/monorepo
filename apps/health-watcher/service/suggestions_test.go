package service

import (
	"context"
	"testing"

	"apps/health-watcher/domain"
	"apps/health-watcher/port"
)

// fakeLOINC knows one LOINC code and describes it.
type fakeLOINC struct{}

func (fakeLOINC) Name(_ context.Context, code string) (string, bool, error) {
	if code == "1-8" {
		return "Test analyte [Mass/volume] in Serum", true, nil
	}
	return "", false, nil
}

func (fakeLOINC) About(_ context.Context, code string) (domain.MetricAbout, bool, error) {
	return domain.MetricAbout{Quote: "A test analyte test measures the analyte.", Source: domain.AboutSource, Title: "Test Analyte",
		URL: "https://medlineplus.gov/lab-tests/test-analyte/", Fetched: "2026-10-23"}, true, nil
}

func unknownLab() domain.ReadingOutput {
	out := labReading()
	out.Observations = []domain.ReadingObservation{
		{Page: 1, Analyte: "GR-ABS", Value: "4.4", Unit: "G/L", Confidence: 97},
		{Page: 1, Analyte: "Chất thử", Value: "12", Unit: "mg/L", Confidence: 97},
		{Page: 1, Analyte: "Fake LOINC test", Value: "3", Unit: "U/L", Confidence: 97},
		{Page: 1, Analyte: "Ghi chú", Value: "xem trang sau", Confidence: 97},
		{Page: 1, Analyte: "Không rõ", Value: "1", Unit: "", Confidence: 97},
	}
	return out
}

func unknownAnswers() domain.SuggestOutput {
	return domain.SuggestOutput{Suggestions: []domain.SuggestAnswer{
		{Name: "GR-ABS", Action: "link", MetricCode: "granulocytes_abs", Reason: "Granulocyte count."},
		{Name: "Chất thử", Action: "new", NewName: "Test analyte", NewNameVI: "Chất thử", LOINC: "1-8", Unit: "mg/L",
			Category: string(domain.CatEnzymes), Reason: "A test the catalog lacks."},
		{Name: "Fake LOINC test", Action: "new", NewName: "Other analyte", LOINC: "99999-9", Unit: "U/L",
			Category: string(domain.CatEnzymes), Reason: "Another test."},
		{Name: "Ghi chú", Action: "none", Reason: "A note."},
		{Name: "Không rõ", Action: "new", NewName: "Unknown", Category: "nonsense", Reason: "Bad category."},
	}}
}

func TestGivenNamesNoMetricKnowsWhenFiledThenTheAppSuggestsAndAPersonDecides(t *testing.T) {
	e := newEnv(t)
	e.pipeline.LOINC = fakeLOINC{}
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	out := unknownLab()
	e.reader.out, e.reader.suggest = &out, unknownAnswers()
	doc := e.filedLab(sc, person, "lab")
	e.drain()

	if len(e.reader.suggestCalls) != 1 || len(e.reader.suggestCalls[0]) != 5 {
		t.Fatalf("suggest calls = %+v", e.reader.suggestCalls)
	}
	if got := e.reader.suggestCalls[0][0]; got != (domain.UnknownName{Key: "gr abs", Analyte: "GR-ABS", Unit: "G/L"}) {
		t.Fatalf("first name = %+v", got)
	}
	list, err := e.clinical.Suggestions(e.ctx, sc)
	if err != nil || len(list) != 3 {
		t.Fatalf("suggestions = %+v, %v", list, err)
	}
	byName := map[string]domain.MetricSuggestion{}
	for _, s := range list {
		byName[s.Analyte] = s
		if s.Values != 1 || s.Model != "test-model" {
			t.Fatalf("suggestion = %+v", s)
		}
	}
	gra, test, fake := byName["GR-ABS"], byName["Chất thử"], byName["Fake LOINC test"]
	if gra.Kind != domain.SuggestLink || gra.MetricCode != "granulocytes_abs" {
		t.Fatalf("GR-ABS = %+v", gra)
	}
	if p := test.Proposal; p == nil || p.LOINC != "1-8" || p.LOINCName == "" || p.About == nil {
		t.Fatalf("Chất thử proposal = %+v", p)
	}
	if p := fake.Proposal; p == nil || p.LOINC != "" || p.About != nil {
		t.Fatalf("an unknown LOINC code is kept: %+v", p)
	}
	overview, err := e.clinical.PersonMetrics(e.ctx, sc, person.ID)
	if err != nil || overview.Suggestions != 3 {
		t.Fatalf("overview suggestions = %d, %v", overview.Suggestions, err)
	}

	// Approving a link links the value, audited as the person's change.
	if _, err := e.clinical.ApproveSuggestion(e.ctx, sc, gra.ID, SuggestionEdit{}); err != nil {
		t.Fatal(err)
	}
	codes := e.codes(sc, doc.Document.ID)
	if codes["GR-ABS"] != "granulocytes_abs" {
		t.Fatalf("GR-ABS = %q", codes["GR-ABS"])
	}
	hist, err := e.clinical.History(e.ctx, sc, domain.AuditObservation, doc.Observations[0].ID)
	if err != nil || len(hist) != 1 || hist[0].UserID != sc.UserID {
		t.Fatalf("history = %+v, %v", hist, err)
	}
	if _, err := e.clinical.ApproveSuggestion(e.ctx, sc, gra.ID, SuggestionEdit{}); err == nil {
		t.Fatal("a suggestion was approved twice")
	}

	// Approving a new metric, renamed, adds it to the family's catalog.
	unit := "mg/dL"
	if _, err := e.clinical.ApproveSuggestion(e.ctx, sc, test.ID, SuggestionEdit{Name: "Test analyte X", Unit: &unit}); err != nil {
		t.Fatal(err)
	}
	code := domain.CustomMetricCode("Test analyte X")
	if codes := e.codes(sc, doc.Document.ID); codes["Chất thử"] != code {
		t.Fatalf("Chất thử = %q", codes["Chất thử"])
	}
	trend, err := e.clinical.PersonMetric(e.ctx, sc, person.ID, code)
	if err != nil || !trend.Metric.Custom || trend.Metric.Unit != "mg/dL" || trend.Metric.LOINC != "1-8" || trend.About == nil ||
		len(trend.Points) != 1 {
		t.Fatalf("trend = %+v, %v", trend, err)
	}
	metrics, err := e.clinical.Metrics(e.ctx, sc)
	if err != nil || metrics[len(metrics)-1].Code != code {
		t.Fatalf("catalog ends with %+v, %v", metrics[len(metrics)-1], err)
	}

	// Rejecting keeps the value under its printed name.
	if _, err := e.clinical.RejectSuggestion(e.ctx, sc, fake.ID); err != nil {
		t.Fatal(err)
	}
	if list, err := e.clinical.Suggestions(e.ctx, sc); err != nil || len(list) != 0 {
		t.Fatalf("pending = %+v, %v", list, err)
	}

	// The next paper with the same names links them; only the name with a
	// dropped answer is asked about again.
	doc2 := e.filedLab(sc, person, "lab-2")
	e.drain()
	codes = e.codes(sc, doc2.Document.ID)
	if codes["GR-ABS"] != "granulocytes_abs" || codes["Chất thử"] != code || codes["Fake LOINC test"] != "" {
		t.Fatalf("second paper codes = %+v", codes)
	}
	if n := len(e.reader.suggestCalls); n != 2 || len(e.reader.suggestCalls[1]) != 1 || e.reader.suggestCalls[1][0].Analyte != "Không rõ" {
		t.Fatalf("suggest calls = %+v", e.reader.suggestCalls)
	}
}

func TestGivenAViewerWhenDecidingASuggestionThenItIsForbidden(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	out := unknownLab()
	e.reader.out, e.reader.suggest = &out, unknownAnswers()
	e.filedLab(sc, person, "lab")
	e.drain()
	list, err := e.clinical.Suggestions(e.ctx, sc)
	if err != nil || len(list) == 0 {
		t.Fatalf("suggestions = %+v, %v", list, err)
	}
	viewer := e.join(sc, "viewer@example.test", "viewer")
	if _, err := e.clinical.ApproveSuggestion(e.ctx, viewer, list[0].ID, SuggestionEdit{}); err == nil {
		t.Fatal("a viewer approved a suggestion")
	}
	if _, err := e.clinical.RejectSuggestion(e.ctx, viewer, list[0].ID); err == nil {
		t.Fatal("a viewer rejected a suggestion")
	}
}

// A link made by hand for "GRAN#" under the old key also matched "GRAN%";
// re-keying keeps it for "GRAN#" alone.
func TestGivenALinkUnderTheOldKeyWhenRekeyedThenCountAndPercentAreApart(t *testing.T) {
	e := newEnv(t)
	sc := e.setup()
	person := e.person(sc, "Nguyễn Văn Mẫu")
	out := labReading()
	out.Observations = []domain.ReadingObservation{{Page: 1, Analyte: "GRAN#", Value: "4.4", Unit: "G/L", Confidence: 97}}
	e.reader.out = &out
	doc := e.filedLab(sc, person, "lab")
	if _, err := e.clinical.LinkMetric(e.ctx, sc, doc.Observations[0].ID, "granulocytes_abs"); err != nil {
		t.Fatal(err)
	}
	var providerID string
	if err := e.store.InTx(e.ctx, func(r port.Repos) error {
		pr, err := r.ProviderByKey(e.ctx, sc.TenantID, domain.FoldKey("Bệnh viện Đa khoa Mẫu"))
		if err != nil {
			return err
		}
		providerID = pr.ID
		if err := r.DeleteAnalyteLink(e.ctx, sc.TenantID, pr.ID, "gran abs"); err != nil {
			return err
		}
		return r.SetAnalyteLink(e.ctx, sc.TenantID, pr.ID, "gran", "granulocytes_abs")
	}); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := e.clinical.RekeyAnalyteLinks(e.ctx); err != nil {
			t.Fatal(err)
		}
	}
	links, err := e.store.AnalyteLinks(e.ctx, sc.TenantID, providerID)
	if err != nil || len(links) != 1 || links["gran abs"] != "granulocytes_abs" {
		t.Fatalf("links = %+v, %v", links, err)
	}

	out.Observations = []domain.ReadingObservation{{Page: 1, Analyte: "GRAN%", Value: "60", Unit: "%", Confidence: 97}}
	doc2 := e.filedLab(sc, person, "lab-2")
	if doc2.Observations[0].MetricCode == "granulocytes_abs" {
		t.Fatal("GRAN% took the GRAN# link")
	}
}

// codes maps a document's printed names to their metric codes.
func (e *env) codes(sc domain.Scope, documentID string) map[string]string {
	e.t.Helper()
	doc, err := e.clinical.Document(e.ctx, sc, documentID)
	if err != nil {
		e.t.Fatal(err)
	}
	out := map[string]string{}
	for _, o := range doc.Observations {
		out[o.Analyte] = o.MetricCode
	}
	return out
}
