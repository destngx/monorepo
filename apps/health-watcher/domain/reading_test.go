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

	fs, _, _ := out.CleanFindings(2, today, nil)
	if len(fs) != 2 || fs[0].ICD10 != "E78.5" || fs[1].Kind != "note" || fs[1].ICD10 != "" {
		t.Fatalf("findings = %+v", fs)
	}
}

func TestGivenPageTitlesFromTheModelWhenCleaningThenEachPageGetsOneTrimmedTitle(t *testing.T) {
	long := strings.Repeat("x", MaxPageTitle+10)
	got := CleanPageTitles([]ReadingPage{
		{Page: 2, Title: "  Siêu âm ổ bụng  "},
		{Page: 1, Title: "Tổng phân tích tế bào máu"},
		{Page: 2, Title: "A second title for page 2"},
		{Page: 0, Title: "No such page"},
		{Page: 9, Title: "No such page"},
		{Page: 3, Title: long},
	}, 4)
	want := []string{"Tổng phân tích tế bào máu", "Siêu âm ổ bụng", long[:MaxPageTitle], ""}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("titles = %q, want %q", got, want)
	}
}

func TestGivenTheReadingSchemaWhenEncodedThenEveryObjectIsStrict(t *testing.T) {
	for _, schema := range []map[string]any{ReadingSchema(), PageTitlesSchema()} {
		assertStrict(t, schema)
	}
	raw, err := json.Marshal(ReadingSchema())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"imaging_film"`) || !strings.Contains(string(raw), `"pages"`) {
		t.Fatal("document kinds or page titles missing from the schema")
	}
}

func assertStrict(t *testing.T, schema map[string]any) {
	t.Helper()
	raw, err := json.Marshal(schema)
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
}

func TestGivenABloodPressurePairWhenCleaningThenItSplitsIntoSystolicAndDiastolic(t *testing.T) {
	out := ReadingOutput{Observations: []ReadingObservation{
		{Page: 1, Analyte: "Huyết áp", Value: "130/85", Unit: "mmHg", Reference: "120/80", Flag: "H", Confidence: 70},
		{Page: 1, Analyte: "Blood pressure", Value: "118 / 76", Confidence: 90},
		{Page: 1, Analyte: "Huyết áp", Value: "130/85", Unit: "cmHg"},
		{Page: 1, Analyte: "Tỷ lệ A/G", Value: "1/2"},
	}}
	obs, pages, confs := out.CleanObservations(1, today, DefaultCatalog)
	if len(obs) != 6 || len(pages) != 6 || len(confs) != 6 {
		t.Fatalf("observations = %+v", obs)
	}
	want := []ObservationInput{
		{Analyte: "Huyết áp tâm thu", Value: "130", Unit: "mmHg", Reference: "120", Flag: "H", MetricCode: "bp_systolic"},
		{Analyte: "Huyết áp tâm trương", Value: "85", Unit: "mmHg", Reference: "80", Flag: "H", MetricCode: "bp_diastolic"},
		{Analyte: "Systolic blood pressure", Value: "118", Unit: "mmHg", MetricCode: "bp_systolic"},
		{Analyte: "Diastolic blood pressure", Value: "76", Unit: "mmHg", MetricCode: "bp_diastolic"},
	}
	for i, w := range want {
		if obs[i] != w {
			t.Errorf("observation %d = %+v, want %+v", i, obs[i], w)
		}
	}
	if confs[0] != 70 || confs[1] != 70 || obs[4].Value != "130/85" || obs[5].Value != "1/2" {
		t.Fatalf("observations = %+v, confidences = %v", obs, confs)
	}
}

func TestGivenAnICD10CodeWithOCRMisreadsWhenCleaningThenDigitsAreRestored(t *testing.T) {
	cases := map[string]string{
		"JO2.9": "J02.9", "j02.9": "J02.9", "E1l.9": "E11.9", "I1O": "I10", "J02,9": "J02.9", " K 29.7 ": "K29.7",
		"080.0": "O80.0", "O80": "O80", "not a code": "", "J0": "", "": "",
	}
	for in, want := range cases {
		if got := CleanICD10(in); got != want {
			t.Errorf("CleanICD10(%q) = %q, want %q", in, got, want)
		}
	}
	out := ReadingOutput{Findings: []ReadingFinding{{Page: 1, Kind: "diagnosis", Text: "Viêm họng cấp", ICD10: "JO2.9"}}}
	if fs, _, _ := out.CleanFindings(1, "2026-10-23", nil); len(fs) != 1 || fs[0].ICD10 != "J02.9" {
		t.Fatalf("findings = %+v", fs)
	}
}

func TestGivenACheckupReadingWhenCleaningFindingsThenExamsAdviceAndTheFullImagingReportAreKept(t *testing.T) {
	out := ReadingOutput{Findings: []ReadingFinding{
		{Page: 1, Kind: "imaging_report", Section: "Chụp X quang tim phổi thẳng",
			Text: "MÔ TẢ HÌNH ẢNH\n• Phổi phải:  đồng nhất.\n\nKẾT LUẬN\nHình ảnh X quang tim phổi hiện tại không thấy bất thường.", Confidence: 90},
		{Page: 2, Kind: "exam", Section: "Khám Tai mũi họng", Text: "Viêm amydan mạn", Confidence: 85},
		{Page: 2, Kind: "advice", Text: "Khám định kỳ chuyên khoa Mắt, tai mũi họng.", Confidence: 85},
	}}
	fs, pages, _ := out.CleanFindings(2, today, nil)
	if len(fs) != 3 || fs[0].Kind != "imaging_report" || fs[1].Kind != "exam" || fs[2].Kind != "advice" {
		t.Fatalf("findings = %+v", fs)
	}
	if fs[0].Section != "Chụp X quang tim phổi thẳng" || !strings.Contains(fs[0].Text, "• Phổi phải: đồng nhất.\nKẾT LUẬN\n") {
		t.Fatalf("imaging report = %+v", fs[0])
	}
	if fs[1].Section != "Khám Tai mũi họng" || pages[1] != 2 {
		t.Fatalf("exam = %+v, page %d", fs[1], pages[1])
	}
}

func TestGivenAVaccinationRecordWhenCleaningFindingsThenEachDoseKeepsItsVaccineAndAFutureDoseIsDue(t *testing.T) {
	out := ReadingOutput{Findings: []ReadingFinding{
		{Page: 1, Kind: "vaccination", Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Vaccine X", Dose: "1ml", Date: "2025-01-15", Text: "Lần 1: 15/01/2025", Confidence: 80},
		{Page: 1, Kind: "vaccination", Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Vaccine X", Dose: "1ml", Date: "2027-06-15", Text: "Lần 3: 15/06/2027", Confidence: 80},
		{Page: 1, Kind: "vaccination", Drug: "Vaccine X", Date: "sometime", Text: "Lần 2", Confidence: 80},
	}}
	fs, _, confs := out.CleanFindings(1, "2026-10-23", nil)
	if len(fs) != 3 {
		t.Fatalf("findings = %+v", fs)
	}
	if fs[0].Kind != "vaccination" || fs[0].Drug != "Vaccine X" || fs[0].Dose != "1ml" || fs[0].Date != "2025-01-15" {
		t.Fatalf("given dose = %+v", fs[0])
	}
	if confs[0] >= 70 || confs[1] >= 70 {
		t.Fatalf("a vaccine name no earlier dose confirms is not flagged: %v", confs)
	}
	due := fs[1]
	if due.Kind != "advice" || due.Date != "2027-06-15" || due.Drug != "" || due.Text != "Vaccine X 1ml\nLần 3: 15/06/2027" || due.Section != "Viêm gan Siêu vi B (Hepatitis B)" {
		t.Fatalf("future dose = %+v", due)
	}
	if fs[2].Kind != "vaccination" || fs[2].Date != "" {
		t.Fatalf("unreadable date = %+v", fs[2])
	}
}

func TestGivenAnEarlierApprovedDoseWhenCleaningVaccinationsThenItsVaccineNameIsUsed(t *testing.T) {
	earlier := LatestVaccines([]Finding{
		{Kind: "vaccination", Status: StatusApproved, Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Vaccine Old", Date: "2025-01-01"},
		{Kind: "vaccination", Status: StatusApproved, Section: "Viêm gan siêu vi B (Hepatitis B)", Drug: "Vaccine X", Date: "2025-01-15"},
		{Kind: "vaccination", Status: StatusDraft, Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Vaccine Draft", Date: "2026-10-01"},
		{Kind: "vaccination", Status: StatusApproved, Section: "Cúm (Influenza)", Drug: "", Date: "2026-10-01"},
	})
	out := ReadingOutput{Findings: []ReadingFinding{
		{Page: 1, Kind: "vaccination", Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Misread", Date: "2025-02-15", Text: "Lần 2: 15/02/2025", Confidence: 95},
		{Page: 1, Kind: "vaccination", Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "vaccine x", Date: "2025-02-15", Text: "Lần 2: 15/02/2025", Confidence: 95},
		{Page: 1, Kind: "vaccination", Section: "Viêm gan Siêu vi B (Hepatitis B)", Drug: "Misread", Dose: "1ml", Date: "2027-06-15", Text: "Lần 3: 15/06/2027", Confidence: 95},
		{Page: 1, Kind: "vaccination", Section: "Cúm (Influenza)", Drug: "Vaccine F", Date: "2025-02-15", Text: "Lần 1", Confidence: 95},
	}}
	fs, _, confs := out.CleanFindings(1, "2026-10-23", earlier)
	if fs[0].Drug != "Vaccine X" || confs[0] >= 70 {
		t.Fatalf("a name unlike the earlier dose = %+v, %d", fs[0], confs[0])
	}
	if fs[1].Drug != "Vaccine X" || confs[1] != 95 {
		t.Fatalf("a name the earlier dose confirms = %+v, %d", fs[1], confs[1])
	}
	if fs[2].Kind != "advice" || fs[2].Text != "Vaccine X 1ml\nLần 3: 15/06/2027" {
		t.Fatalf("a due dose = %+v", fs[2])
	}
	if fs[3].Drug != "Vaccine F" || confs[3] >= 70 {
		t.Fatalf("a disease without an earlier named dose = %+v, %d", fs[3], confs[3])
	}
}

func TestGivenPageTextWhenCheckingReadableThenOnlyPagesWithEnoughTextAreSent(t *testing.T) {
	if Readable("R  L\n 2026 ") {
		t.Fatal("a film photo's few letters count as readable")
	}
	if !Readable("KẾT QUẢ CHỤP X-QUANG\nKẾT LUẬN: không thấy bất thường") {
		t.Fatal("a report page is not readable")
	}
}

func TestGivenAResultPrintedWithItsReadingWhenCleaningThenOnlyTheNumberIsKept(t *testing.T) {
	cases := map[string]string{
		"Negative 0.794":    "0.794",
		"0,794 (Âm tính)":   "0,794",
		"Dương tính: 12.5":  "12.5",
		"Non-reactive 0.12": "0.12",
		"Âm tính":           "Âm tính",
		"< 0.5":             "< 0.5",
		"Negative < 0.5":    "Negative < 0.5",
		"1/80 Positive":     "1/80 Positive",
		"Negative 0.1 0.2":  "Negative 0.1 0.2",
		"6.4":               "6.4",
		"Trace 0.3":         "Trace 0.3",
	}
	for value, want := range cases {
		out := ReadingOutput{Observations: []ReadingObservation{{Page: 1, Analyte: "HBsAg miễn dịch tự động", Value: value, Unit: "S/Co"}}}
		obs, _, _ := out.CleanObservations(1, today, DefaultCatalog)
		if len(obs) != 1 || obs[0].Value != want {
			t.Errorf("%q = %+v, want %q", value, obs, want)
		}
	}
}
