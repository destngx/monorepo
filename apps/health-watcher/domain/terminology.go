package domain

import (
	"regexp"
	"strings"
)

// Code system URIs the records map to (D9, docs/health-watcher/DATA-MODEL.md §3).
const (
	SystemLOINC                     = "http://loinc.org"
	SystemUCUM                      = "http://unitsofmeasure.org"
	SystemICD10                     = "http://hl7.org/fhir/sid/icd-10"
	SystemObservationInterpretation = "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation"
	SystemObservationCategory       = "http://terminology.hl7.org/CodeSystem/observation-category"
	SystemActCode                   = "http://terminology.hl7.org/CodeSystem/v3-ActCode"
	SystemOrganizationType          = "http://terminology.hl7.org/CodeSystem/organization-type"
	SystemConditionClinical         = "http://terminology.hl7.org/CodeSystem/condition-clinical"
	SystemConditionVerification     = "http://terminology.hl7.org/CodeSystem/condition-ver-status"
	SystemDiagnosticServiceSection  = "http://terminology.hl7.org/CodeSystem/v2-0074"
	// SystemApp prefixes the app's own code systems, such as visit types
	// and document kinds: SystemApp + "/visit-type".
	SystemApp = "https://health-watcher.local/fhir/CodeSystem"
)

// Coding is one code from a code system, as FHIR's Coding.
type Coding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// CodeableConcept is a concept given as codes plus the text a person or
// paper used, as FHIR's CodeableConcept.
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// AppCoding is a code from one of the app's own code systems.
func AppCoding(system, code string) Coding {
	return Coding{System: SystemApp + "/" + system, Code: code}
}

// FHIRResource names the FHIR R4 resource type a record maps to.
type FHIRResource string

const (
	ResourcePatient            FHIRResource = "Patient"
	ResourceOrganization       FHIRResource = "Organization"
	ResourceEncounter          FHIRResource = "Encounter"
	ResourceDocumentReference  FHIRResource = "DocumentReference"
	ResourceDiagnosticReport   FHIRResource = "DiagnosticReport"
	ResourceObservation        FHIRResource = "Observation"
	ResourceCondition          FHIRResource = "Condition"
	ResourceMedicationRequest  FHIRResource = "MedicationRequest"
	ResourceProcedure          FHIRResource = "Procedure"
	ResourceAllergyIntolerance FHIRResource = "AllergyIntolerance"
	ResourceImmunization       FHIRResource = "Immunization"
	ResourceCarePlan           FHIRResource = "CarePlan"
	// ResourceNone marks a finding that is part of another resource: an
	// imaging report is its document's DiagnosticReport.conclusion and a
	// note is an annotation on the encounter.
	ResourceNone FHIRResource = ""
)

// FindingResource is the FHIR resource a finding kind maps to.
func FindingResource(kind FindingKind) FHIRResource {
	switch kind {
	case "diagnosis":
		return ResourceCondition
	case "exam":
		return ResourceObservation
	case "prescription":
		return ResourceMedicationRequest
	case "procedure":
		return ResourceProcedure
	case "allergy":
		return ResourceAllergyIntolerance
	case "vaccination":
		return ResourceImmunization
	case "advice":
		return ResourceCarePlan
	}
	return ResourceNone
}

// Gender is the Patient.gender code for a person's sex.
func Gender(s Sex) string {
	if s == SexUnknown {
		return "unknown"
	}
	return string(s)
}

// OrganizationType is the Organization.type of a provider: the HL7 code
// and the app's finer kind.
func OrganizationType(kind ProviderKind) CodeableConcept {
	code := Coding{System: SystemOrganizationType, Code: "prov", Display: "Healthcare Provider"}
	if kind == ProviderOther {
		code = Coding{System: SystemOrganizationType, Code: "other", Display: "Other"}
	}
	return CodeableConcept{Coding: []Coding{code, AppCoding("provider-kind", string(kind))}}
}

// EncounterClass is the Encounter.class of a visit type.
func EncounterClass(t VisitType) Coding {
	switch t {
	case "inpatient":
		return Coding{System: SystemActCode, Code: "IMP", Display: "inpatient encounter"}
	case "emergency":
		return Coding{System: SystemActCode, Code: "EMER", Display: "emergency"}
	}
	return Coding{System: SystemActCode, Code: "AMB", Display: "ambulatory"}
}

// documentTypes maps each document kind to a LOINC document code. Codes
// were checked against the NLM Clinical Tables LOINC service on 2026-10-22.
var documentTypes = map[DocumentKind]Coding{
	"lab_result":        {SystemLOINC, "11502-2", "Laboratory report"},
	"imaging_report":    {SystemLOINC, "18748-4", "Diagnostic imaging study"},
	"imaging_film":      {SystemLOINC, "18748-4", "Diagnostic imaging study"},
	"prescription":      {SystemLOINC, "57833-6", "Prescription for medication"},
	"discharge_summary": {SystemLOINC, "18842-5", "Discharge summary"},
	"dental_record":     {SystemLOINC, "28618-7", "Dentistry Note"},
	"record_book_page":  {SystemLOINC, "11506-3", "Progress note"},
	"invoice":           {SystemLOINC, "94093-2", "Itemized bill"},
	"other":             {SystemLOINC, "34109-9", "Note"},
}

// DocumentType is the DocumentReference.type of a document kind.
func DocumentType(kind DocumentKind) CodeableConcept {
	code, ok := documentTypes[kind]
	if !ok {
		code = documentTypes["other"]
	}
	return CodeableConcept{Coding: []Coding{code, AppCoding("document-kind", string(kind))}}
}

// ReportCategory is the DiagnosticReport.category of a document kind, and
// false for kinds that make no report.
func ReportCategory(kind DocumentKind) (Coding, bool) {
	switch kind {
	case "lab_result":
		return Coding{System: SystemDiagnosticServiceSection, Code: "LAB", Display: "Laboratory"}, true
	case "imaging_report", "imaging_film":
		return Coding{System: SystemDiagnosticServiceSection, Code: "RAD", Display: "Radiology"}, true
	}
	return Coding{}, false
}

// ObservationCategory is the Observation.category of a value: vital-signs
// for a vital sign in the catalog, laboratory otherwise.
func ObservationCategory(m *Metric) Coding {
	if m != nil && m.Category == CatVitals {
		return Coding{System: SystemObservationCategory, Code: "vital-signs", Display: "Vital Signs"}
	}
	return Coding{System: SystemObservationCategory, Code: "laboratory", Display: "Laboratory"}
}

// ExamCategory is the Observation.category of an exam finding.
var ExamCategory = Coding{System: SystemObservationCategory, Code: "exam", Display: "Exam"}

// Interpretation is the Observation.interpretation of the paper's flag,
// and false when the paper printed none.
func Interpretation(flag string) (Coding, bool) {
	switch flag {
	case "H":
		return Coding{System: SystemObservationInterpretation, Code: "H", Display: "High"}, true
	case "L":
		return Coding{System: SystemObservationInterpretation, Code: "L", Display: "Low"}, true
	case "*":
		return Coding{System: SystemObservationInterpretation, Code: "A", Display: "Abnormal"}, true
	}
	return Coding{}, false
}

// ucumUnits maps the catalog's unit spellings (NormalizeUnit) to UCUM.
var ucumUnits = map[string]string{
	"%": "%", "g/L": "g/L", "g/dL": "g/dL", "mg/L": "mg/L", "mg/dL": "mg/dL", "µg/L": "ug/L", "µg/dL": "ug/dL",
	"µg/mL": "ug/mL", "ng/mL": "ng/mL", "ng/dL": "ng/dL", "pg/mL": "pg/mL", "pg": "pg", "fL": "fL",
	"mmol/L": "mmol/L", "µmol/L": "umol/L", "nmol/L": "nmol/L", "pmol/L": "pmol/L", "mmol/mol": "mmol/mol",
	"mEq/L": "meq/L", "U/L": "U/L", "mIU/L": "m[IU]/L", "µIU/mL": "u[IU]/mL", "mIU/mL": "m[IU]/mL", "IU/mL": "[IU]/mL",
	"L/L": "L/L", "mg/g": "mg/g", "mg/mmol": "mg/mmol", "mL/min/1.73m²": "mL/min/{1.73_m2}", "mm/h": "mm/h", "s": "s",
	"G/L": "10*9/L", "T/L": "10*12/L", "mmHg": "mm[Hg]", "kg": "kg", "lb": "[lb_av]", "cm": "cm", "m": "m",
	"in": "[in_i]", "kg/m²": "kg/m2", "/min": "/min", "°C": "Cel", "°F": "[degF]",
	// Cut-off ratios have no UCUM unit; LOINC writes them as annotations.
	"S/CO": "{s_co_ratio}", "COI": "{COI}",
}

// UCUMUnit is the UCUM code of a printed unit, and false when the unit is
// not one the catalog knows. A wrong UCUM code would let another system
// convert a value wrongly, so unknown units keep only their printed text.
func UCUMUnit(printed string) (string, bool) {
	code, ok := ucumUnits[NormalizeUnit(printed)]
	return code, ok
}

var comparatorValue = regexp.MustCompile(`^(<=|>=|≤|≥|<|>)\s*([+-]?(?:\d+(?:[.,]\d+)?|[.,]\d+))$`)

// ParseComparator reads a value printed with a comparator (`< 0.5`) as
// Quantity.comparator and its number. ok is false for any other value.
func ParseComparator(value string) (comparator string, number float64, ok bool) {
	m := comparatorValue.FindStringSubmatch(strings.TrimSpace(value))
	if m == nil {
		return "", 0, false
	}
	n := ParseNumber(m[2])
	if n == nil {
		return "", 0, false
	}
	comparator = strings.NewReplacer("≤", "<=", "≥", ">=").Replace(m[1])
	return comparator, *n, true
}

var provisionalDiagnosis = regexp.MustCompile(`^(?:td|t d|theo doi|nghi)\b`)

// ConditionVerification is the Condition.verificationStatus of a diagnosis:
// provisional when the paper marks it as under observation (`TD: ...`,
// `theo dõi ...`, `nghi ...`), confirmed otherwise.
func ConditionVerification(text string) Coding {
	if provisionalDiagnosis.MatchString(strings.ToLower(FoldKey(text))) {
		return Coding{System: SystemConditionVerification, Code: "provisional", Display: "Provisional"}
	}
	return Coding{System: SystemConditionVerification, Code: "confirmed", Display: "Confirmed"}
}
