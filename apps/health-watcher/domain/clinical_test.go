package domain

import (
	"errors"
	"strings"
	"testing"
)

func fp(v float64) *float64 { return &v }

func eqPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestGivenPrintedResultsWhenParsingNumbersThenOnlyPlainNumbersCount(t *testing.T) {
	cases := map[string]*float64{"6.4": fp(6.4), "6,4": fp(6.4), " 88 ": fp(88), "-1.5": fp(-1.5), "< 0.5": nil, "Âm tính": nil, "6.4 H": nil, "": nil}
	for in, want := range cases {
		if got := ParseNumber(in); !eqPtr(got, want) {
			t.Errorf("ParseNumber(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestGivenPrintedRangesWhenParsingThenBoundsComeFromThePaper(t *testing.T) {
	cases := []struct {
		in        string
		low, high *float64
	}{
		{"3.9 - 6.1", fp(3.9), fp(6.1)}, {"3,9–6,1", fp(3.9), fp(6.1)}, {"< 5.2", nil, fp(5.2)},
		{"> 1.03", fp(1.03), nil}, {"≤ 40", nil, fp(40)}, {"Âm tính", nil, nil}, {"", nil, nil},
	}
	for _, tc := range cases {
		low, high := ParseRange(tc.in)
		if !eqPtr(low, tc.low) || !eqPtr(high, tc.high) {
			t.Errorf("ParseRange(%q) = %v, %v", tc.in, low, high)
		}
	}
}

func TestGivenPrintedFlagsWhenParsingThenTheyBecomeHLOrStar(t *testing.T) {
	for in, want := range map[string]string{"H": "H", "cao": "H", "↑": "H", "l": "L", "Thấp": "L", "*": "*", "": ""} {
		if got, err := ParseFlag(in); err != nil || got != want {
			t.Errorf("ParseFlag(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseFlag("very high"); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown flag accepted: %v", err)
	}
}

func TestGivenAnObservationInputWhenValidatingThenItIsCleanedAndChecked(t *testing.T) {
	in, err := ObservationInput{Analyte: " Glucose  (đói) ", Value: "6.4", Unit: "mmol/L", Flag: "h", SampleDate: "2025-03-14", MetricCode: "glucose"}.
		Validate(today, DefaultCatalog)
	if err != nil || in.Analyte != "Glucose (đói)" || in.Flag != "H" {
		t.Fatalf("Validate = %+v, %v", in, err)
	}
	bad := map[string]ObservationInput{
		"no name":      {Value: "1"},
		"no value":     {Analyte: "A"},
		"future date":  {Analyte: "A", Value: "1", SampleDate: "2027-01-01"},
		"unknown code": {Analyte: "A", Value: "1", MetricCode: "nope"},
		"unknown flag": {Analyte: "A", Value: "1", Flag: "x"},
	}
	for name, in := range bad {
		if _, err := in.Validate(today, DefaultCatalog); !errors.Is(err, ErrValidation) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestGivenAFindingInputWhenValidatingThenDrugFieldsStayOnPrescriptionsAndVaccinations(t *testing.T) {
	in, err := FindingInput{Kind: "diagnosis", Text: "Đái tháo đường type 2", ICD10: "e11.9", Drug: "Metformin"}.Validate()
	if err != nil || in.ICD10 != "E11.9" || in.Drug != "" {
		t.Fatalf("Validate = %+v, %v", in, err)
	}
	rx, err := FindingInput{Kind: "prescription", Drug: "Metformin 500mg", Dose: "1 viên", Frequency: "2 lần/ngày", Duration: "30 ngày"}.Validate()
	if err != nil || rx.Drug != "Metformin 500mg" {
		t.Fatalf("prescription = %+v, %v", rx, err)
	}
	vax, err := FindingInput{Kind: "vaccination", Drug: "Vaccine X", Dose: "1ml", Frequency: "x", Duration: "y", Date: "2025-01-15"}.Validate()
	if err != nil || vax.Drug != "Vaccine X" || vax.Dose != "1ml" || vax.Frequency != "" || vax.Duration != "" || vax.Date != "2025-01-15" {
		t.Fatalf("vaccination = %+v, %v", vax, err)
	}
	if _, err := (FindingInput{Kind: "vaccination", Drug: "Vaccine X", Date: "15/01/2025"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad date accepted: %v", err)
	}
	if _, err := (FindingInput{Kind: "diagnosis", Text: "x", ICD10: "diabetes"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad ICD-10 accepted: %v", err)
	}
	if _, err := (FindingInput{Kind: "surgery", Text: "x"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("unknown kind accepted: %v", err)
	}
}

func TestGivenAnExamOrAdviceFindingWhenValidatingThenSectionAndLinesAreKept(t *testing.T) {
	in, err := FindingInput{Kind: "exam", Section: "  Khám   Mắt ", Text: "Không kính MP: 1/10.\n  Có kính MP: 10/10. "}.Validate()
	if err != nil || in.Section != "Khám Mắt" || in.Text != "Không kính MP: 1/10.\nCó kính MP: 10/10." {
		t.Fatalf("exam = %+v, %v", in, err)
	}
	if _, err := (FindingInput{Kind: "advice", Text: "Khám định kỳ chuyên khoa Mắt."}).Validate(); err != nil {
		t.Fatalf("advice refused: %v", err)
	}
	if _, err := (FindingInput{Kind: "exam", Section: strings.Repeat("a", 121), Text: "x"}).Validate(); !errors.Is(err, ErrValidation) {
		t.Fatalf("long section accepted: %v", err)
	}
}
