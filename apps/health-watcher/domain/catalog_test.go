package domain

import (
	"math"
	"testing"
)

func TestGivenPrintedAnalyteNamesWhenMatchingThenEachFindsItsCatalogEntry(t *testing.T) {
	cases := []struct{ name, unit, want string }{
		{"Glucose (đói)", "mmol/L", "glucose"},
		{"Đường huyết", "", "glucose"},
		{"HbA1c", "%", "hba1c"},
		{"Cholesterol toàn phần", "mmol/L", "cholesterol_total"},
		{"HDL-Cholesterol", "mmol/L", "hdl"},
		{"LDL-C (tính toán)", "mmol/L", "ldl"},
		{"Triglycerid", "", "triglycerides"},
		{"Ure", "mmol/L", "urea"},
		{"Creatinin", "µmol/L", "creatinine"},
		{"eGFR (CKD-EPI)", "mL/ph/1.73m²", "egfr"},
		{"AST (GOT)", "U/L", "ast"},
		{"ALT (GPT)", "U/L", "alt"},
		{"GGT", "U/L", "ggt"},
		{"Acid Uric", "umol/L", "uric_acid"},
		{"WBC (Số lượng bạch cầu)", "G/L", "wbc"},
		{"RBC (Số lượng hồng cầu)", "T/L", "rbc"},
		{"HGB (Huyết sắc tố)", "g/L", "hemoglobin"},
		{"PLT (Số lượng tiểu cầu)", "G/L", "platelets"},
		{"Protein niệu", "", "urine_protein"},
		{"Glucose niệu", "", "urine_glucose"},
		{"Nitrit", "", "urine_nitrite"},
		{"NEU%", "%", "neutrophils_pct"},
		{"NEU#", "G/L", "neutrophils_abs"},
		{"Trung tính", "%", "neutrophils_pct"},
		{"Trung tính", "G/L", "neutrophils_abs"},
		{"Glucose máu tĩnh mạch", "mmol/L", "glucose"},
	}
	for _, tc := range cases {
		m, ok := DefaultCatalog.Match(tc.name, tc.unit, "1.0")
		if !ok || m.Code != tc.want {
			t.Errorf("Match(%q, %q) = %q, %v; want %q", tc.name, tc.unit, m.Code, ok, tc.want)
		}
	}
}

// A complete blood count names each white cell type after the white cell
// count (`Số lượng bạch cầu đoạn`), and a urine strip prints `Bạch cầu`
// with no word for urine: neither may join the white cell count's trend.
func TestGivenNamesThatStartWithACountsNameWhenMatchingThenTheCountIsNotPicked(t *testing.T) {
	cases := []struct{ name, unit, value, want string }{
		{"Số lượng bạch cầu (WBC)", "G/L", "7.7", "wbc"},
		{"Số lượng bạch cầu đoạn (GRAN#)", "G/L", "4.4", "granulocytes_abs"},
		{"Tỷ lệ % bạch cầu đoạn (GRAN%)", "%", "58.0", "granulocytes_pct"},
		{"Số lượng bạch cầu Mid", "G/L", "0.7", "mid_abs"},
		{"Bạch cầu ưa acid", "", "0.2", "eosinophils_pct"},
		{"Số lượng hồng cầu lưới", "%", "1.1", ""},
		{"Bạch cầu (que thử)", "BC/µL", "Âm tính", "urine_leukocytes"},
		{"Bạch cầu", "G/L", "7.1", "wbc"},
		{"Tỷ lệ % bạch cầu Lympho", "%", "31.0", "lymphocytes_pct"},
		{"Tỷ lệ % bạch cầu Mid", "%", "7.5", "mid_pct"},
	}
	for _, tc := range cases {
		m, ok := DefaultCatalog.Match(tc.name, tc.unit, tc.value)
		if m.Code != tc.want || ok != (tc.want != "") {
			t.Errorf("Match(%q, %q, %q) = %q, %v; want %q", tc.name, tc.unit, tc.value, m.Code, ok, tc.want)
		}
	}
}

func TestGivenUnknownOrAmbiguousNamesWhenMatchingThenNothingIsGuessed(t *testing.T) {
	for _, tc := range []struct{ name, unit string }{
		{"Protein", "g/L"}, // a blood unit, but only the urine test is called just `Protein`
		{"Trung tính", ""},
		{"Xét nghiệm lạ", "mg/dL"},
		{"", ""},
	} {
		if m, ok := DefaultCatalog.Match(tc.name, tc.unit, ""); ok {
			t.Errorf("Match(%q) = %q, want no match", tc.name, m.Code)
		}
	}
}

func TestGivenPrintedUnitsWhenNormalizingThenSpellingsAndCaseRulesAgree(t *testing.T) {
	cases := map[string]string{
		"umol/L": "µmol/L", "μmol/l": "µmol/L", " mmol/l ": "mmol/L", "G/L": "G/L", "g/L": "g/L", "g/l": "g/L",
		"10^9/L": "G/L", "x10*9/L": "G/L", "K/uL": "G/L", "T/l": "T/L", "10^12/L": "T/L",
		"mL/ph/1.73m²": "mL/min/1.73m²", "mL/phút/1.73m2": "mL/min/1.73m²", "IU/L": "U/L", "mg/dl": "mg/dL",
		"lần/phút": "/min", "°C": "°C", "weird": "weird", "": "",
	}
	for in, want := range cases {
		if got := NormalizeUnit(in); got != want {
			t.Errorf("NormalizeUnit(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGivenValuesInOtherUnitsWhenConvertingThenTheCanonicalValueUsesTheClinicalFactors(t *testing.T) {
	cases := []struct {
		code, unit string
		in, want   float64
	}{
		{"glucose", "mg/dL", 100, 5.550622},
		{"cholesterol_total", "mg/dL", 200, 5.172},
		{"creatinine", "mg/dL", 1, 88.42},
		{"hemoglobin", "g/dL", 14.8, 148},
		{"hba1c", "mmol/mol", 48, 6.54304},
		{"temperature", "°F", 98.6, 37},
		{"wbc", "10^9/L", 7.85, 7.85},
		{"glucose", "mmol/L", 6.4, 6.4},
	}
	for _, tc := range cases {
		m, _ := DefaultCatalog.Metric(tc.code)
		got, ok := m.ToCanonical(tc.in, tc.unit)
		if !ok || math.Abs(got-tc.want) > 1e-3 {
			t.Errorf("%s %v %s = %v, %v; want %v", tc.code, tc.in, tc.unit, got, ok, tc.want)
		}
	}
	m, _ := DefaultCatalog.Metric("glucose")
	if _, ok := m.ToCanonical(1, "U/L"); ok {
		t.Fatal("glucose converted from an enzyme unit")
	}
}

func TestGivenTheCatalogWhenCheckingEntriesThenCodesAreUniqueAndConversionsTargetOtherUnits(t *testing.T) {
	if n := len(DefaultCatalog.Metrics()); n < 60 {
		t.Fatalf("catalog has %d entries, want about 60 or more", n)
	}
	seen := map[string]bool{}
	for _, m := range DefaultCatalog.Metrics() {
		if seen[m.Code] || m.LOINC == "" || m.Name == "" || m.NameVI == "" || m.Category == "" {
			t.Errorf("bad entry %+v", m)
		}
		seen[m.Code] = true
		if m.Unit != NormalizeUnit(m.Unit) {
			t.Errorf("%s: canonical unit %q is not in normal form", m.Code, m.Unit)
		}
		for _, c := range m.Conversions {
			if c.From == m.Unit || c.From != NormalizeUnit(c.From) || c.Factor == 0 {
				t.Errorf("%s: bad conversion %+v", m.Code, c)
			}
		}
	}
}
