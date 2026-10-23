package domain

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// RecordStatus says whether an observation or finding counts yet
// (HW-READ-4). Only approved ones appear in charts and reviews.
type RecordStatus string

const (
	StatusDraft    RecordStatus = "draft"
	StatusApproved RecordStatus = "approved"
	StatusRejected RecordStatus = "rejected"
)

// RecordSource says how an observation or finding was made (D9): read by
// AI from a document, typed by a person, or imported from another system.
// It becomes the FHIR Provenance activity.
type RecordSource string

const (
	SourceReading RecordSource = "reading"
	SourceManual  RecordSource = "manual"
	SourceImport  RecordSource = "import"
)

// Provenance is the version and origin every observation and finding
// carries (DATA-MODEL.md §5). Version starts at 1 and each edit of an
// approved record raises it. SourceDetail is the reading's model or the
// import id. ApprovedBy and ApprovedAt are empty for drafts and for records
// approved before D9; a manual entry is approved by its author when saved.
type Provenance struct {
	Version      int          `json:"version"`
	Source       RecordSource `json:"source"`
	SourceDetail string       `json:"sourceDetail,omitempty"`
	ApprovedBy   string       `json:"approvedBy,omitempty"`
	ApprovedAt   *time.Time   `json:"approvedAt,omitempty"`
}

// Observation is one measured value as printed on a paper (HW-REC-3).
type Observation struct {
	ID         string       `json:"id"`
	Status     RecordStatus `json:"status"`
	ItemID     string       `json:"itemId,omitempty"`
	DocumentID string       `json:"documentId,omitempty"`
	PersonID   string       `json:"personId,omitempty"`
	PageID     string       `json:"pageId,omitempty"`
	PageNumber int          `json:"pageNumber,omitempty"`
	Analyte    string       `json:"analyte"`
	MetricCode string       `json:"metricCode,omitempty"`
	Value      string       `json:"value"`
	ValueNum   *float64     `json:"valueNum,omitempty"`
	Unit       string       `json:"unit"`
	RefText    string       `json:"refText"`
	RefLow     *float64     `json:"refLow,omitempty"`
	RefHigh    *float64     `json:"refHigh,omitempty"`
	Flag       string       `json:"flag"`
	SampleDate string       `json:"sampleDate,omitempty"`
	// Confidence is the reader's confidence, 0-100; 100 for manual entries.
	Confidence int `json:"confidence"`
	Provenance
	// ExportGaps lists what a complete export still needs (RecordExportGaps).
	ExportGaps []ExportGap `json:"exportGaps,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// ObservationInput is a new or changed observation, from the AI reading or
// typed by hand (HW-READ-7).
type ObservationInput struct {
	Analyte    string `json:"analyte"`
	MetricCode string `json:"metricCode"`
	Value      string `json:"value"`
	Unit       string `json:"unit"`
	Reference  string `json:"reference"`
	Flag       string `json:"flag"`
	SampleDate string `json:"sampleDate"`
	PageID     string `json:"pageId"`
}

// Validate cleans the input. A catalog code must exist; empty means unlinked.
func (in ObservationInput) Validate(today string, catalog *Catalog) (ObservationInput, error) {
	in.Analyte = CleanText(in.Analyte)
	in.Value = CleanText(in.Value)
	in.Unit = CleanText(in.Unit)
	in.Reference = CleanText(in.Reference)
	if in.Analyte == "" {
		return in, Invalid("the test name is required")
	}
	if in.Value == "" {
		return in, Invalid("the result is required")
	}
	for name, v := range map[string]string{"test name": in.Analyte, "result": in.Value, "unit": in.Unit, "reference range": in.Reference} {
		if utf8.RuneCountInString(v) > 120 {
			return in, Invalid("%s is longer than 120 characters", name)
		}
	}
	flag, err := ParseFlag(in.Flag)
	if err != nil {
		return in, err
	}
	in.Flag = flag
	if in.SampleDate != "" {
		if in.SampleDate, err = Date(in.SampleDate); err != nil {
			return in, Invalid("sample date must be YYYY-MM-DD")
		}
		if in.SampleDate > today {
			return in, Invalid("sample date is in the future")
		}
	}
	if in.MetricCode != "" {
		if _, ok := catalog.Metric(in.MetricCode); !ok {
			return in, Invalid("unknown metric %q", in.MetricCode)
		}
	}
	return in, nil
}

var (
	plainNumber = regexp.MustCompile(`^[+-]?(?:\d+(?:[.,]\d+)?|[.,]\d+)$`)
	rangeBoth   = regexp.MustCompile(`^([+-]?\d+(?:[.,]\d+)?)\s*(?:-|–|—|~|đến|to)\s*([+-]?\d+(?:[.,]\d+)?)$`)
	rangeBelow  = regexp.MustCompile(`^(?:<|≤|<=|nhỏ hơn|dưới)\s*(\d+(?:[.,]\d+)?)$`)
	rangeAbove  = regexp.MustCompile(`^(?:>|≥|>=|lớn hơn|trên)\s*(\d+(?:[.,]\d+)?)$`)
)

// ParseNumber reads a printed result as a number. A decimal comma is
// accepted (`6,4`); a value with a comparator or words (`< 0.5`, `Âm
// tính`) is not a number and gives nil.
func ParseNumber(s string) *float64 {
	s = strings.TrimSpace(s)
	if !plainNumber.MatchString(s) {
		return nil
	}
	v, err := strconv.ParseFloat(strings.Replace(s, ",", ".", 1), 64)
	if err != nil {
		return nil
	}
	return &v
}

// ParseRange reads a printed reference range: `3.9 - 6.1`, `< 5.2`,
// `> 1.03`. Anything else (`Âm tính`) gives no bounds and stays as text.
func ParseRange(s string) (low, high *float64) {
	s = strings.ToLower(CleanText(s))
	if m := rangeBoth.FindStringSubmatch(s); m != nil {
		return ParseNumber(m[1]), ParseNumber(m[2])
	}
	if m := rangeBelow.FindStringSubmatch(s); m != nil {
		return nil, ParseNumber(m[1])
	}
	if m := rangeAbove.FindStringSubmatch(s); m != nil {
		return ParseNumber(m[1]), nil
	}
	return nil, nil
}

// ParseFlag reads the paper's own flag (HW-MET-5): H, L or `*`, from the
// spellings labs print. The app never computes a flag itself.
func ParseFlag(s string) (string, error) {
	switch strings.ToLower(CleanText(s)) {
	case "":
		return "", nil
	case "h", "high", "cao", "tăng", "↑", "+", "hh":
		return "H", nil
	case "l", "low", "thấp", "giảm", "↓", "-", "ll":
		return "L", nil
	case "*", "**", "!", "bất thường", "abnormal", "a":
		return "*", nil
	}
	return "", Invalid("flag must be H, L, * or empty")
}

// FindingKind is the kind of a finding (HW-REC-4).
type FindingKind string

// FindingKinds lists every finding kind in display order. An exam is one
// specialty's or body system's examination result and comment; advice is
// what the doctor tells the patient to do next.
var FindingKinds = []FindingKind{"diagnosis", "exam", "prescription", "procedure", "imaging_report", "advice", "allergy", "vaccination", "note"}

// ParseFindingKind validates a finding kind; empty means note.
func ParseFindingKind(s string) (FindingKind, error) {
	if s == "" {
		return "note", nil
	}
	for _, k := range FindingKinds {
		if string(k) == s {
			return k, nil
		}
	}
	return "", Invalid("unknown finding kind %q", s)
}

// Finding is a diagnosis, exam result, prescription, procedure, imaging
// report, advice, allergy, vaccination or note as written on a paper
// (HW-REC-4).
type Finding struct {
	ID         string       `json:"id"`
	Status     RecordStatus `json:"status"`
	ItemID     string       `json:"itemId,omitempty"`
	DocumentID string       `json:"documentId,omitempty"`
	PersonID   string       `json:"personId,omitempty"`
	PageID     string       `json:"pageId,omitempty"`
	PageNumber int          `json:"pageNumber,omitempty"`
	Kind       FindingKind  `json:"kind"`
	// Section is the heading the finding was written under: the specialty
	// or body system of an exam, the study of an imaging report.
	Section   string `json:"section,omitempty"`
	Text      string `json:"text"`
	ICD10     string `json:"icd10,omitempty"`
	Drug      string `json:"drug,omitempty"`
	Dose      string `json:"dose,omitempty"`
	Frequency string `json:"frequency,omitempty"`
	Duration  string `json:"duration,omitempty"`
	// Date is the day written beside the finding, such as the day of one
	// vaccine dose or the day a next dose is due; "" means the visit date.
	Date       string `json:"date,omitempty"`
	Confidence int    `json:"confidence"`
	Provenance
	// ExportGaps lists what a complete export still needs (RecordExportGaps).
	ExportGaps []ExportGap `json:"exportGaps,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
	UpdatedAt  time.Time   `json:"updatedAt"`
}

// FindingInput is a new or changed finding.
type FindingInput struct {
	Kind      string `json:"kind"`
	Section   string `json:"section"`
	Text      string `json:"text"`
	ICD10     string `json:"icd10"`
	Drug      string `json:"drug"`
	Dose      string `json:"dose"`
	Frequency string `json:"frequency"`
	Duration  string `json:"duration"`
	Date      string `json:"date"`
	PageID    string `json:"pageId"`
}

var icd10Code = regexp.MustCompile(`^[A-Z][0-9]{2}(?:\.[0-9A-Z]{1,4})?$`)

// CleanICD10 fixes the usual OCR misreads in an ICD-10 code read from paper:
// the letter O for zero and I or l for one in the digit places, zero for O
// in the letter place, a comma for the dot and stray spaces. A string that
// still is not a code gives "".
func CleanICD10(s string) string {
	r := []rune(strings.ToUpper(strings.ReplaceAll(strings.Join(strings.Fields(s), ""), ",", ".")))
	if len(r) < 3 {
		return ""
	}
	switch r[0] {
	case '0':
		r[0] = 'O'
	case '1':
		r[0] = 'I'
	}
	for i := 1; i < 3; i++ {
		switch r[i] {
		case 'O':
			r[i] = '0'
		case 'I', 'L':
			r[i] = '1'
		}
	}
	if code := string(r); icd10Code.MatchString(code) {
		return code
	}
	return ""
}

// Validate cleans the input. Drug fields belong to prescriptions only.
func (in FindingInput) Validate() (FindingInput, error) {
	kind, err := ParseFindingKind(in.Kind)
	if err != nil {
		return in, err
	}
	in.Kind = string(kind)
	in.Section = CleanText(in.Section)
	in.Text = CleanLines(in.Text)
	in.ICD10 = strings.ToUpper(strings.TrimSpace(in.ICD10))
	in.Drug, in.Dose, in.Frequency, in.Duration = CleanText(in.Drug), CleanText(in.Dose), CleanText(in.Frequency), CleanText(in.Duration)
	// A vaccination keeps its vaccine in Drug and the amount given in Dose.
	switch kind {
	case "prescription":
	case "vaccination":
		in.Frequency, in.Duration = "", ""
	default:
		in.Drug, in.Dose, in.Frequency, in.Duration = "", "", "", ""
	}
	if in.Date = strings.TrimSpace(in.Date); in.Date != "" {
		if _, err := Date(in.Date); err != nil {
			return in, err
		}
	}
	if in.Text == "" && in.Drug == "" {
		return in, Invalid("the finding needs some text")
	}
	if utf8.RuneCountInString(in.Text) > 4000 {
		return in, Invalid("text is longer than 4000 characters")
	}
	if utf8.RuneCountInString(in.Section) > 120 {
		return in, Invalid("section is longer than 120 characters")
	}
	for name, v := range map[string]string{"drug": in.Drug, "dose": in.Dose, "frequency": in.Frequency, "duration": in.Duration} {
		if utf8.RuneCountInString(v) > 200 {
			return in, Invalid("%s is longer than 200 characters", name)
		}
	}
	if in.ICD10 != "" && !icd10Code.MatchString(in.ICD10) {
		return in, Invalid("ICD-10 code must look like E11 or E11.9")
	}
	return in, nil
}

// AuditEntity is the kind of record an audit entry is about.
type AuditEntity string

const (
	AuditObservation AuditEntity = "observation"
	AuditFinding     AuditEntity = "finding"
)

// AuditEntry keeps the old value of an approved record someone changed or
// deleted (HW-REC-6).
type AuditEntry struct {
	ID       string      `json:"id"`
	Entity   AuditEntity `json:"entity"`
	EntityID string      `json:"entityId"`
	Action   string      `json:"action"`
	// Version is the version the old value had.
	Version int       `json:"version"`
	UserID  string    `json:"userId"`
	At      time.Time `json:"at"`
	OldJSON string    `json:"old"`
}
