package domain

import (
	"errors"
	"testing"
)

func f(v float64) *float64 { return &v }

func metric(t *testing.T, code string) Metric {
	t.Helper()
	m, ok := DefaultCatalog.Metric(code)
	if !ok {
		t.Fatalf("no metric %q", code)
	}
	return m
}

func obs(date, value, unit string, num, low, high *float64, flag, documentID string) PersonObservation {
	return PersonObservation{Observation: Observation{ID: "o-" + date, DocumentID: documentID, SampleDate: date, Value: value,
		ValueNum: num, Unit: unit, RefLow: low, RefHigh: high, Flag: flag}}
}

func TestGivenAGlucoseInMgdLWhenPlottedThenValueAndPaperRangeAreInMmolL(t *testing.T) {
	p := TrendPointOf(metric(t, "glucose"), obs("2025-01-02", "126", "mg/dL", f(126), f(70), f(110), "H", "d1"))
	if p.Value == nil || *p.Value != 6.993783 || *p.RefLow != 3.885435 || *p.RefHigh != 6.105684 {
		t.Fatalf("point = %+v", p)
	}
	if p.Printed != "126" || p.PrintedUnit != "mg/dL" || p.Flag != "H" || p.Source != SourcePaper {
		t.Fatalf("point = %+v", p)
	}
}

func TestGivenAnUnknownUnitOrTextResultWhenPlottedThenThePointHasNoValueOrRange(t *testing.T) {
	p := TrendPointOf(metric(t, "glucose"), obs("2025-01-02", "6.1", "furlongs", f(6.1), f(3.9), f(6.1), "", "d1"))
	if p.Value != nil || p.RefLow != nil || p.RefHigh != nil || p.Printed != "6.1" {
		t.Fatalf("unknown unit = %+v", p)
	}
	p = TrendPointOf(metric(t, "glucose"), obs("2025-01-02", "Âm tính", "mmol/L", nil, f(3.9), f(6.1), "", "d1"))
	if p.Value != nil || p.RefLow != nil {
		t.Fatalf("text = %+v", p)
	}
}

func TestGivenNoDocumentWhenPlottedThenThePointIsTypedIn(t *testing.T) {
	if p := TrendPointOf(metric(t, "weight"), obs("2025-01-02", "60", "kg", f(60), nil, nil, "", "")); p.Source != SourceTyped {
		t.Fatalf("source = %q", p.Source)
	}
}

func TestGivenWeightsAndHeightsWhenBMIIsComputedThenEachWeightUsesTheLatestEarlierHeight(t *testing.T) {
	heights := []TrendPoint{{Date: "2024-01-01", Value: f(160)}, {Date: "2025-06-01", Value: f(170)}}
	weights := []TrendPoint{
		{Date: "2023-12-31", Value: f(60)}, // before any height
		{Date: "2024-03-01", Value: f(64)},
		{Date: "2025-06-01", Value: f(72.25)},
		{Date: "2025-07-01", Value: f(70)}, // a printed BMI exists that day
	}
	printed := []TrendPoint{{Date: "2025-07-01", Value: f(24.1)}}
	got := ComputeBMI(weights, heights, printed, "1980-05-05")
	if len(got) != 2 || got[0].Date != "2024-03-01" || *got[0].Value != 25 || got[1].Date != "2025-06-01" || *got[1].Value != 25 {
		t.Fatalf("bmi = %+v", got)
	}
	if got[0].Source != SourceComputed || got[0].Printed != "25" || got[0].PrintedUnit != "kg/m²" || got[0].RefLow != nil || got[0].Flag != "" {
		t.Fatalf("bmi point = %+v", got[0])
	}
}

func TestGivenAChildWhenTheHeightIsOldThenNoBMIIsComputed(t *testing.T) {
	heights := []TrendPoint{{Date: "2024-01-01", Value: f(120)}}
	weights := []TrendPoint{{Date: "2024-03-01", Value: f(25)}, {Date: "2024-09-01", Value: f(27)}}
	got := ComputeBMI(weights, heights, nil, "2017-02-02")
	if len(got) != 1 || got[0].Date != "2024-03-01" {
		t.Fatalf("bmi = %+v", got)
	}
}

func TestGivenPointsWhenTheOverviewIsBuiltThenItHasTheLatestTheChangeAndRecentNumbers(t *testing.T) {
	points := []TrendPoint{{Date: "2024-01-01", Value: f(5.2)}, {Date: "2024-06-01", Printed: "Âm tính"}, {Date: "2025-01-01", Value: f(6.3), Flag: "H"}}
	item, ok := OverviewOf(metric(t, "glucose"), points)
	if !ok || item.Latest.Date != "2025-01-01" || item.Latest.Flag != "H" || item.Count != 3 || item.Previous.Date != "2024-06-01" {
		t.Fatalf("item = %+v", item)
	}
	if item.Change != nil {
		t.Fatalf("change across a text result = %v", *item.Change)
	}
	if len(item.Recent) != 2 || item.Recent[0].Date != "2024-01-01" {
		t.Fatalf("recent = %+v", item.Recent)
	}
	item, _ = OverviewOf(metric(t, "glucose"), []TrendPoint{points[0], points[2]})
	if item.Change == nil || *item.Change != 1.1 {
		t.Fatalf("change = %v", item.Change)
	}
	if _, ok := OverviewOf(metric(t, "glucose"), nil); ok {
		t.Fatal("empty overview item")
	}
}

func TestGivenTypedVitalsWhenValidatedThenUnitsDefaultAndBadValuesAreRejected(t *testing.T) {
	in := MeasurementInput{Date: "2025-03-01", Values: []MeasurementValue{
		{MetricCode: "weight", Value: "61,5"}, {MetricCode: "temperature", Value: "99.1", Unit: "°f"}, {MetricCode: "heart_rate", Value: " "},
	}}
	got, err := in.Validate("2025-03-02", DefaultCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Values) != 2 || got.Values[0].Unit != "kg" || got.Values[1].Unit != "°F" {
		t.Fatalf("values = %+v", got.Values)
	}
	bad := []MeasurementInput{
		{Date: "2025-03-03", Values: in.Values},
		{Date: "2025-03-01", Values: []MeasurementValue{{MetricCode: "glucose", Value: "5"}}},
		{Date: "2025-03-01", Values: []MeasurementValue{{MetricCode: "weight", Value: "-1"}}},
		{Date: "2025-03-01", Values: []MeasurementValue{{MetricCode: "weight", Value: "60", Unit: "mmHg"}}},
		{Date: "2025-03-01", Values: []MeasurementValue{{MetricCode: "weight", Value: "60"}, {MetricCode: "weight", Value: "61"}}},
		{Date: "2025-03-01"},
	}
	for i, b := range bad {
		if _, err := b.Validate("2025-03-02", DefaultCatalog); !errors.Is(err, ErrValidation) {
			t.Errorf("bad[%d] = %v", i, err)
		}
	}
}

func TestGivenNamesTheCatalogDoesNotKnowWhenCodedThenSpellingsOfOneNameShareACode(t *testing.T) {
	cases := map[string]string{
		"Tỷ lệ % bạch cầu Mid": "printed-ty-le-bach-cau-mid",
		"TY LE % BACH CAU MID": "printed-ty-le-bach-cau-mid",
		"Mid#":                 "printed-mid-abs",
		"Mid%":                 "printed-mid-pct",
		"Chỉ số lạ (XYZ)":      "printed-chi-so-la-xyz",
		"???":                  "printed-unnamed",
	}
	for in, want := range cases {
		if got := PrintedMetricCode(in); got != want {
			t.Errorf("PrintedMetricCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGivenValuesOfAnUnknownNameWhenPlottedThenTheLatestNameAndUnitWinAndOtherUnitsAreOnlyListed(t *testing.T) {
	old := obs("2025-01-02", "1.2", "mg/l", f(1.2), f(0), f(5), "", "d1")
	old.Analyte = "Chỉ số lạ"
	text := obs("2025-03-01", "Âm tính", "", nil, nil, nil, "", "d2")
	text.Analyte = "Chỉ số lạ"
	other := obs("2025-05-01", "3", "U/L", f(3), nil, nil, "", "d3")
	other.Analyte = "Chỉ số lạ"
	latest := obs("2025-06-01", "2.5", "mg/L", f(2.5), nil, f(5), "H", "d4")
	latest.Analyte = "CHỈ SỐ LẠ"
	all := []PersonObservation{old, text, other, latest}

	m := PrintedMetric(PrintedMetricCode(latest.Analyte), all)
	if m.Name != "CHỈ SỐ LẠ" || m.Unit != "mg/L" || m.Category != CatPrinted || !m.Printed || m.LOINC != "" {
		t.Fatalf("metric = %+v", m)
	}
	if p := TrendPointOf(m, old); p.Value == nil || *p.Value != 1.2 || *p.RefHigh != 5 {
		t.Fatalf("same unit, other spelling = %+v", p)
	}
	if p := TrendPointOf(m, other); p.Value != nil || p.Printed != "3" {
		t.Fatalf("other unit = %+v", p)
	}
	if p := TrendPointOf(m, text); p.Value != nil || p.Printed != "Âm tính" {
		t.Fatalf("text = %+v", p)
	}
}
