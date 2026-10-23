package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGivenUnlinkedValuesWhenGroupedThenEachFoldedNameOnceWithItsLatestSpellingAndUnit(t *testing.T) {
	obs := []Observation{
		{Analyte: "Chỉ số lạ", Unit: "mg/l"},
		{Analyte: "Mid#", Unit: "G/L"},
		{Analyte: "Mid%", Unit: "%"},
		{Analyte: "CHỈ SỐ LẠ", Unit: "mg/L"},
		{Analyte: "Đã hỏi", Unit: ""},
	}
	got := UnknownNames(obs, map[string]bool{AnalyteKey("Đã hỏi"): true})
	want := []UnknownName{{Key: "chi so la", Analyte: "CHỈ SỐ LẠ", Unit: "mg/L"}, {Key: "mid abs", Analyte: "Mid#", Unit: "G/L"},
		{Key: "mid pct", Analyte: "Mid%", Unit: "%"}}
	if len(got) != len(want) {
		t.Fatalf("names = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestGivenAnAIAnswerWhenCleanedThenOnlyUsableSuggestionsForNamesAskedAboutAreKept(t *testing.T) {
	names := []UnknownName{
		{Key: "gra abs", Analyte: "GRA#", Unit: "G/L"},
		{Key: "chi so moi", Analyte: "Chỉ số mới", Unit: "mg/L"},
		{Key: "ghi chu", Analyte: "Ghi chú"},
		{Key: "sai ma", Analyte: "Sai mã"},
		{Key: "sai nhom", Analyte: "Sai nhóm"},
		{Key: "ma loinc la", Analyte: "Mã LOINC lạ"},
	}
	out := SuggestOutput{Suggestions: []SuggestAnswer{
		{Name: "GRA#", Action: "link", MetricCode: "granulocytes_abs", Reason: "Granulocyte count."},
		{Name: "GRA#", Action: "new", NewName: "A second answer", Category: "blood_count"},
		{Name: "Chỉ số mới", Action: "new", NewName: "New test", NewNameVI: "Chỉ số mới", LOINC: "1234-5", Unit: "mg/l", Category: "enzymes"},
		{Name: "Ghi chú", Action: "none", Reason: "A note."},
		{Name: "Sai mã", Action: "link", MetricCode: "no_such_metric"},
		{Name: "Sai nhóm", Action: "new", NewName: "Odd", Category: "made_up"},
		{Name: "Mã LOINC lạ", Action: "new", NewName: "Odd code", LOINC: "not a code", Category: "liver"},
		{Name: "Never asked", Action: "link", MetricCode: "wbc"},
	}}
	got := out.Clean(names, DefaultCatalog)
	if len(got) != 4 {
		t.Fatalf("suggestions = %+v", got)
	}
	if g := got[0]; g.Kind != SuggestLink || g.MetricCode != "granulocytes_abs" || g.AnalyteKey != "gra abs" || g.Status != SuggestionPending {
		t.Errorf("link = %+v", g)
	}
	if p := got[1].Proposal; got[1].Kind != SuggestNew || p == nil || p.LOINC != "1234-5" || p.Unit != "mg/L" || p.Category != CatEnzymes {
		t.Errorf("new = %+v %+v", got[1], p)
	}
	if got[2].Kind != SuggestNone || got[2].Status != SuggestionRejected {
		t.Errorf("none = %+v", got[2])
	}
	if p := got[3].Proposal; p == nil || p.LOINC != "" {
		t.Errorf("bad LOINC kept = %+v", p)
	}
}

func TestGivenTheSuggestSchemaWhenEncodedThenItIsStrictAndListsTheCategories(t *testing.T) {
	assertStrict(t, SuggestSchema())
	raw, _ := json.Marshal(SuggestSchema())
	if !strings.Contains(string(raw), `"blood_count"`) || strings.Contains(string(raw), `"printed"`) {
		t.Fatalf("schema = %s", raw)
	}
}

func TestGivenACustomMetricWhenAddedToTheCatalogThenItMatchesByItsNamesAndTheShippedOnesStay(t *testing.T) {
	custom := Metric{Code: CustomMetricCode("New test"), Name: "New test", Category: CatEnzymes, Unit: "mg/L",
		Aliases: []string{"Chỉ số mới"}, Custom: true}
	c := DefaultCatalog.With([]Metric{custom})
	if custom.Code != "custom-new-test" || c.Version != DefaultCatalog.Version {
		t.Fatalf("code %q, version %q", custom.Code, c.Version)
	}
	if m, ok := c.Match("Chỉ số mới", "mg/L", "1.0"); !ok || m.Code != custom.Code {
		t.Fatalf("match = %q, %v", m.Code, ok)
	}
	if _, ok := c.Metric("wbc"); !ok || len(c.Metrics()) != len(DefaultCatalog.Metrics())+1 {
		t.Fatal("shipped metrics lost")
	}
	if _, ok := DefaultCatalog.Metric(custom.Code); ok {
		t.Fatal("the shipped catalog changed")
	}
}
