package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGivenAModelAnswerWhenCleaningThenEnumsDatesPagesAndFlagsAreChecked(t *testing.T) {
	out := ReadingOutput{
		Document: ReadingDocument{Kind: "lab_result", Title: "Phiếu xét nghiệm", Date: "2025-03-14", VisitType: "lab",
			ProviderName: "BỆNH VIỆN ĐA KHOA MẪU", ProviderKind: "hospital", Confidence: 120},
		Observations: []ReadingObservation{
			{Page: 1, Analyte: "Glucose (đói)", Value: "6.4", Unit: "mmol/L", Reference: "3.9 - 6.1", Flag: "H", SampleDate: "2025-03-14", Confidence: 95},
			{Page: 9, Analyte: "Protein niệu", Value: "Âm tính", Reference: "Âm tính", Flag: "?", Confidence: -5},
			{Page: 1, Analyte: "", Value: "1"},
		},
		Findings: []ReadingFinding{
			{Page: 1, Kind: "diagnosis", Text: "Rối loạn lipid máu", ICD10: "E78.5", Confidence: 80},
			{Page: 1, Kind: "weird", Text: "Tái khám sau 3 tháng", ICD10: "not a code"},
			{Page: 1, Kind: "diagnosis"},
		},
	}
	p := out.Proposal(today)
	if p.Kind != "lab_result" || p.VisitType != "lab" || p.ProviderKind != "hospital" || p.Confidence != 100 || p.Date != "2025-03-14" {
		t.Fatalf("proposal = %+v", p)
	}
	future := ReadingOutput{Document: ReadingDocument{Kind: "x", Date: "2030-01-01", VisitType: "y", ProviderKind: "z"}}.Proposal(today)
	if future.Kind != "other" || future.VisitType != "other" || future.ProviderKind != "other" || future.Date != "" {
		t.Fatalf("bad enums kept: %+v", future)
	}

	obs, pages, confs := out.CleanObservations(2, today, DefaultCatalog)
	if len(obs) != 2 || obs[0].MetricCode != "glucose" || obs[0].Flag != "H" || obs[1].MetricCode != "urine_protein" || obs[1].Flag != "" {
		t.Fatalf("observations = %+v", obs)
	}
	if pages[1] != 1 || confs[1] != 0 {
		t.Fatalf("pages = %v, confidences = %v", pages, confs)
	}

	fs, _, _ := out.CleanFindings(2)
	if len(fs) != 2 || fs[0].ICD10 != "E78.5" || fs[1].Kind != "note" || fs[1].ICD10 != "" {
		t.Fatalf("findings = %+v", fs)
	}
}

func TestGivenTheReadingSchemaWhenEncodedThenEveryObjectIsStrict(t *testing.T) {
	raw, err := json.Marshal(ReadingSchema())
	if err != nil {
		t.Fatal(err)
	}
	var walk func(v any)
	walk = func(v any) {
		switch n := v.(type) {
		case map[string]any:
			if n["type"] == "object" {
				props, _ := n["properties"].(map[string]any)
				req, _ := n["required"].([]any)
				if n["additionalProperties"] != false || len(props) != len(req) {
					t.Errorf("object is not strict: %v", n)
				}
			}
			for _, c := range n {
				walk(c)
			}
		case []any:
			for _, c := range n {
				walk(c)
			}
		}
	}
	var decoded any
	_ = json.Unmarshal(raw, &decoded)
	walk(decoded)
	if !strings.Contains(string(raw), `"imaging_film"`) {
		t.Fatal("document kinds missing from the schema")
	}
}
