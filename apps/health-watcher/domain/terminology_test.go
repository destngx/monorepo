package domain

import "testing"

func TestGivenEveryCatalogUnitWhenMappingToUCUMThenEachHasACode(t *testing.T) {
	for _, m := range DefaultCatalog.Metrics() {
		units := []string{m.Unit}
		for _, c := range m.Conversions {
			units = append(units, c.From)
		}
		for _, u := range units {
			if u == "" {
				continue
			}
			if _, ok := UCUMUnit(u); !ok {
				t.Errorf("metric %s: unit %q has no UCUM code", m.Code, u)
			}
		}
	}
}

func TestGivenPrintedUnitsWhenMappingToUCUMThenCountUnitsKeepTheirCase(t *testing.T) {
	cases := []struct {
		printed, want string
		ok            bool
	}{
		{"G/L", "10*9/L", true},
		{"g/L", "g/L", true},
		{"10^9/L", "10*9/L", true},
		{"T/L", "10*12/L", true},
		{"μmol/l", "umol/L", true},
		{"mL/ph/1.73m²", "mL/min/{1.73_m2}", true},
		{"mmHg", "mm[Hg]", true},
		{"°C", "Cel", true},
		{"IU/L", "U/L", true},
		{"cái", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, ok := UCUMUnit(tc.printed)
		if got != tc.want || ok != tc.ok {
			t.Errorf("UCUMUnit(%q) = %q, %v; want %q, %v", tc.printed, got, ok, tc.want, tc.ok)
		}
	}
}

func TestGivenValuesWithComparatorsWhenParsingThenNumberAndComparatorSplit(t *testing.T) {
	cases := []struct {
		value, cmp string
		num        float64
		ok         bool
	}{
		{"< 0.5", "<", 0.5, true},
		{"<0,5", "<", 0.5, true},
		{">= 90", ">=", 90, true},
		{"≤ 5.2", "<=", 5.2, true},
		{"≥1.03", ">=", 1.03, true},
		{"> 1000", ">", 1000, true},
		{"5.6", "", 0, false},
		{"Âm tính", "", 0, false},
		{"< âm tính", "", 0, false},
	}
	for _, tc := range cases {
		cmp, num, ok := ParseComparator(tc.value)
		if cmp != tc.cmp || num != tc.num || ok != tc.ok {
			t.Errorf("ParseComparator(%q) = %q, %v, %v; want %q, %v, %v", tc.value, cmp, num, ok, tc.cmp, tc.num, tc.ok)
		}
	}
}

func TestGivenDiagnosisTextWhenVerifyingThenUnderObservationIsProvisional(t *testing.T) {
	cases := map[string]string{
		"TD: Viêm phổi":           "provisional",
		"T/D viêm dạ dày":         "provisional",
		"Theo dõi sốt xuất huyết": "provisional",
		"Nghi ngờ lao phổi":       "provisional",
		"Viêm họng cấp":           "confirmed",
		"Tăng huyết áp, theo dõi": "confirmed",
		"Tiểu đường type 2 (E11)": "confirmed",
	}
	for text, want := range cases {
		got := ConditionVerification(text)
		if got.Code != want || got.System != SystemConditionVerification {
			t.Errorf("ConditionVerification(%q) = %+v; want %s", text, got, want)
		}
	}
}

func TestGivenEveryAppKindWhenMappingThenEachHasAStandardCode(t *testing.T) {
	for _, k := range DocumentKinds {
		cc := DocumentType(k)
		if len(cc.Coding) != 2 || cc.Coding[0].System != SystemLOINC || cc.Coding[0].Code == "" {
			t.Errorf("DocumentType(%s) = %+v; want a LOINC code first", k, cc)
		}
	}
	for _, v := range VisitTypes {
		if c := EncounterClass(v); c.System != SystemActCode || c.Code == "" {
			t.Errorf("EncounterClass(%s) = %+v", v, c)
		}
	}
	if EncounterClass("inpatient").Code != "IMP" || EncounterClass("emergency").Code != "EMER" || EncounterClass("dental").Code != "AMB" {
		t.Error("encounter classes: inpatient IMP, emergency EMER, others AMB")
	}
	for _, k := range []ProviderKind{ProviderHospital, ProviderClinic, ProviderDental, ProviderLab, ProviderPharmacy, ProviderOther} {
		if cc := OrganizationType(k); cc.Coding[0].System != SystemOrganizationType || cc.Coding[1].Code != string(k) {
			t.Errorf("OrganizationType(%s) = %+v", k, cc)
		}
	}
	for _, k := range FindingKinds {
		r := FindingResource(k)
		if (r == ResourceNone) != (k == "imaging_report" || k == "note") {
			t.Errorf("FindingResource(%s) = %q", k, r)
		}
	}
	for flag, want := range map[string]string{"H": "H", "L": "L", "*": "A"} {
		if c, ok := Interpretation(flag); !ok || c.Code != want {
			t.Errorf("Interpretation(%q) = %+v, %v; want %s", flag, c, ok, want)
		}
	}
	if _, ok := Interpretation(""); ok {
		t.Error("an empty flag has no interpretation")
	}
	for s, want := range map[Sex]string{SexFemale: "female", SexMale: "male", SexUnknown: "unknown"} {
		if got := Gender(s); got != want {
			t.Errorf("Gender(%q) = %q; want %q", s, got, want)
		}
	}
	vitals, _ := DefaultCatalog.Metric("bp_systolic")
	glucose, _ := DefaultCatalog.Metric("glucose")
	if ObservationCategory(&vitals).Code != "vital-signs" || ObservationCategory(&glucose).Code != "laboratory" || ObservationCategory(nil).Code != "laboratory" {
		t.Error("observation categories: vitals vital-signs, everything else laboratory")
	}
}
