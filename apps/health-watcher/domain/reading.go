package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ReadingStatus is where an inbox item's AI reading is (HW-READ-3, HW-READ-6).
type ReadingStatus string

const (
	// ReadingWaiting items still have files in OCR.
	ReadingWaiting ReadingStatus = "waiting"
	// ReadingOff means AI is not configured; the item is filed by hand.
	ReadingOff ReadingStatus = "off"
	// ReadingNeedsCheck items wait for a person to check the masked pages.
	ReadingNeedsCheck ReadingStatus = "needs_check"
	ReadingQueued     ReadingStatus = "queued"
	ReadingDone       ReadingStatus = "done"
	// ReadingFailed items ran out of attempts; Retry starts again.
	ReadingFailed ReadingStatus = "failed"
	// ReadingSkipped items are never sent: no text (a film photo or a blank
	// page), or a person turned reading off for the item.
	ReadingSkipped ReadingStatus = "skipped"
)

// Reasons an item is skipped.
const (
	SkipNoText = "No text was found, so there is nothing to read: a film photo or a blank page."
	SkipByUser = "Reading was turned off for this item."
)

// MinReadableChars is the least OCR text a page needs before it is sent.
const MinReadableChars = 30

// Readable reports whether a page has enough text to send. A film photo or
// a blank page has almost none, and is never sent (D5).
func Readable(text string) bool {
	n := 0
	for _, r := range text {
		if !unicode.IsSpace(r) {
			if n++; n >= MinReadableChars {
				return true
			}
		}
	}
	return false
}

// MaxReadPages bounds the pages sent in one reading.
const MaxReadPages = 20

// Proposal is the reading's suggestion for filing the item. PersonID comes
// from the local name match or the backlog person, never from the model.
type Proposal struct {
	PersonID     string       `json:"personId,omitempty"`
	Date         string       `json:"date,omitempty"`
	VisitType    VisitType    `json:"visitType,omitempty"`
	ProviderName string       `json:"providerName,omitempty"`
	ProviderKind ProviderKind `json:"providerKind,omitempty"`
	Kind         DocumentKind `json:"kind,omitempty"`
	Title        string       `json:"title,omitempty"`
	Confidence   int          `json:"confidence"`
}

// Reading is the AI reading of one inbox item.
type Reading struct {
	ItemID        string        `json:"itemId"`
	Status        ReadingStatus `json:"status"`
	Reason        string        `json:"reason,omitempty"`
	Model         string        `json:"model,omitempty"`
	Proposal      *Proposal     `json:"proposal,omitempty"`
	Attempts      int           `json:"attempts"`
	NextAttemptAt *time.Time    `json:"nextAttemptAt,omitempty"`
	Error         string        `json:"error,omitempty"`
	PagesSent     int           `json:"pagesSent"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

// ReadingDocument is the model's view of the document as a whole.
type ReadingDocument struct {
	Kind         string `json:"kind"`
	Title        string `json:"title"`
	Date         string `json:"date"`
	VisitType    string `json:"visit_type"`
	ProviderName string `json:"provider_name"`
	ProviderKind string `json:"provider_kind"`
	Confidence   int    `json:"confidence"`
}

// ReadingObservation is one value the model read.
type ReadingObservation struct {
	Page       int    `json:"page"`
	Analyte    string `json:"analyte"`
	Value      string `json:"value"`
	Unit       string `json:"unit"`
	Reference  string `json:"reference"`
	Flag       string `json:"flag"`
	SampleDate string `json:"sample_date"`
	Confidence int    `json:"confidence"`
}

// ReadingFinding is one finding the model read.
type ReadingFinding struct {
	Page       int    `json:"page"`
	Kind       string `json:"kind"`
	Section    string `json:"section"`
	Text       string `json:"text"`
	ICD10      string `json:"icd10"`
	Drug       string `json:"drug"`
	Dose       string `json:"dose"`
	Frequency  string `json:"frequency"`
	Duration   string `json:"duration"`
	Date       string `json:"date"`
	Confidence int    `json:"confidence"`
}

// ReadingPage is the title the model gave one page.
type ReadingPage struct {
	Page  int    `json:"page"`
	Title string `json:"title"`
}

// ReadingOutput is the model's answer, shaped by ReadingSchema.
type ReadingOutput struct {
	Document     ReadingDocument      `json:"document"`
	Pages        []ReadingPage        `json:"pages"`
	Observations []ReadingObservation `json:"observations"`
	Findings     []ReadingFinding     `json:"findings"`
}

// PageTitlesOutput is the answer of a title job, shaped by PageTitlesSchema.
type PageTitlesOutput struct {
	Pages []ReadingPage `json:"pages"`
}

// MaxPageTitle bounds a page title in characters.
const MaxPageTitle = 80

// UntextedPageTitle names a page with too little text to send to the AI,
// such as a film photo or a blank page (D5), so every page has a title.
const UntextedPageTitle = "Photo or blank page"

// CleanPageTitles returns one title per page, 1-based page n at index n-1;
// a page the model skipped, or numbered outside the document, gets "".
// The first title given for a page wins.
func CleanPageTitles(pages []ReadingPage, pageCount int) []string {
	out := make([]string, pageCount)
	for _, p := range pages {
		if p.Page < 1 || p.Page > pageCount || out[p.Page-1] != "" {
			continue
		}
		out[p.Page-1] = cut(p.Title, MaxPageTitle)
	}
	return out
}

// Limits on what one reading may propose.
const (
	MaxReadObservations = 300
	MaxReadFindings     = 100
)

func clampConfidence(c int) int { return max(0, min(100, c)) }

func cut(s string, n int) string {
	s = CleanText(s)
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n])
	}
	return s
}

// cutLines is cut for multi-line text: each line cleaned, blank lines dropped.
func cutLines(s string, n int) string {
	s = CleanLines(s)
	if utf8.RuneCountInString(s) > n {
		return string([]rune(s)[:n])
	}
	return s
}

func cleanDate(s, today string) string {
	d, err := Date(strings.TrimSpace(s))
	if err != nil || d > today {
		return ""
	}
	return d
}

// Proposal returns the document-level suggestion, with every enum checked:
// an unknown value becomes the default rather than an error, since the
// person fixes it on the review page.
func (o ReadingOutput) Proposal(today string) Proposal {
	d := o.Document
	p := Proposal{Date: cleanDate(d.Date, today), Title: cut(d.Title, 120), ProviderName: cut(d.ProviderName, 120),
		Confidence: clampConfidence(d.Confidence)}
	if k, err := ParseDocumentKind(d.Kind); err == nil {
		p.Kind = k
	} else {
		p.Kind = "other"
	}
	if t, err := ParseVisitType(d.VisitType); err == nil {
		p.VisitType = t
	} else {
		p.VisitType = "other"
	}
	if k, err := ParseProviderKind(d.ProviderKind); err == nil {
		p.ProviderKind = k
	} else {
		p.ProviderKind = ProviderOther
	}
	return p
}

// CleanObservations keeps the readable observations, each as a valid input
// with its page number (1-based, within pageCount) and confidence. Flags the
// reader made up are dropped rather than failing the row.
func (o ReadingOutput) CleanObservations(pageCount int, today string, catalog *Catalog) ([]ObservationInput, []int, []int) {
	var ins []ObservationInput
	var pages, confs []int
	for _, r := range o.Observations {
		if len(ins) == MaxReadObservations {
			break
		}
		flag, err := ParseFlag(r.Flag)
		if err != nil {
			flag = ""
		}
		in := ObservationInput{Analyte: cut(r.Analyte, 120), Value: numberOverWord(cut(r.Value, 120)), Unit: cut(r.Unit, 120),
			Reference: cut(r.Reference, 120), Flag: flag, SampleDate: cleanDate(r.SampleDate, today)}
		if in.Analyte == "" || in.Value == "" {
			continue
		}
		parts := splitBloodPressure(in, catalog)
		if parts == nil {
			if m, ok := catalog.Match(in.Analyte, in.Unit, in.Value); ok {
				in.MetricCode = m.Code
			}
			parts = []ObservationInput{in}
		}
		if len(ins)+len(parts) > MaxReadObservations {
			break
		}
		for _, p := range parts {
			ins = append(ins, p)
			pages = append(pages, clampPage(r.Page, pageCount))
			confs = append(confs, clampConfidence(r.Confidence))
		}
	}
	return ins, pages, confs
}

var (
	oneNumber = regexp.MustCompile(`[+-]?\d+(?:[.,]\d+)?`)
	// resultWords are the printed readings of a test with a cut-off, as
	// FoldKey gives them.
	resultWords = map[string]bool{"negative": true, "positive": true, "neg": true, "pos": true, "am tinh": true, "duong tinh": true,
		"reactive": true, "non reactive": true, "nonreactive": true, "phan ung": true, "khong phan ung": true}
)

// numberOverWord keeps only the number of a result printed with its
// reading, such as `Negative 0.794` or `0.794 (Âm tính)`, so it is a
// value a chart can show. Anything else is returned as it is.
func numberOverWord(value string) string {
	nums := oneNumber.FindAllString(value, -1)
	if len(nums) != 1 {
		return value
	}
	// Only the reading may sit beside the number, never a comparator such
	// as `<`, which makes the number a bound.
	rest := strings.Replace(value, nums[0], " ", 1)
	for _, r := range rest {
		if !unicode.IsLetter(r) && !unicode.Is(unicode.Mn, r) && !unicode.IsSpace(r) && !strings.ContainsRune("():-", r) {
			return value
		}
	}
	if !resultWords[FoldKey(rest)] {
		return value
	}
	return nums[0]
}

var pressurePair = regexp.MustCompile(`^([0-9]{2,3})\s*/\s*([0-9]{2,3})$`)

// splitBloodPressure turns a blood pressure written as one pair, such as
// `130/85 mmHg`, into its systolic and diastolic observations so each one
// links to its catalog metric. Anything else gives nil.
func splitBloodPressure(in ObservationInput, catalog *Catalog) []ObservationInput {
	key := FoldKey(in.Analyte)
	vi := strings.Contains(key, "huyet ap") || key == "ha"
	if !vi && !strings.Contains(key, "blood pressure") && key != "bp" {
		return nil
	}
	if in.Unit != "" && FoldKey(in.Unit) != "mmhg" {
		return nil
	}
	v := pressurePair.FindStringSubmatch(in.Value)
	if v == nil {
		return nil
	}
	ref := pressurePair.FindStringSubmatch(in.Reference)
	var out []ObservationInput
	for i, code := range []string{"bp_systolic", "bp_diastolic"} {
		m, ok := catalog.Metric(code)
		if !ok {
			return nil
		}
		p := in
		p.Analyte, p.MetricCode, p.Value, p.Unit = m.Name, m.Code, v[i+1], "mmHg"
		if vi {
			p.Analyte = m.NameVI
		}
		if ref != nil {
			p.Reference = ref[i+1]
		}
		out = append(out, p)
	}
	return out
}

// UncheckedVaccineConfidence is the most confidence a vaccine name read by
// the model gets when no earlier dose confirms it. Vaccine names are often
// faint, slanted stamps the model misreads with high confidence, so the
// review page asks a person to check each one (HW-READ-4).
const UncheckedVaccineConfidence = 60

// LatestVaccines maps each disease, by the folded section of its approved
// doses, to the vaccine of its latest approved dose with a name.
func LatestVaccines(findings []Finding) map[string]string {
	out := map[string]string{}
	dates := map[string]string{}
	for _, f := range findings {
		key := FoldKey(f.Section)
		if f.Kind != "vaccination" || f.Status != StatusApproved || f.Drug == "" || key == "" {
			continue
		}
		if d, ok := dates[key]; !ok || f.Date >= d {
			out[key], dates[key] = f.Drug, f.Date
		}
	}
	return out
}

// CleanFindings keeps the valid findings, with pages and confidences. A
// date that is not a calendar date is dropped.
//
// A vaccination takes the vaccine of the person's latest approved dose for
// the same disease, from vaccines (LatestVaccines); a name no earlier dose
// confirms keeps what the model read and is flagged for a check. A
// vaccination dated after today has not been given yet: record books write
// the next dose's day in advance, so it becomes advice that the dose is
// due, with the vaccine and dose leading its text.
func (o ReadingOutput) CleanFindings(pageCount int, today string, vaccines map[string]string) ([]FindingInput, []int, []int) {
	var ins []FindingInput
	var pages, confs []int
	for _, r := range o.Findings {
		if len(ins) == MaxReadFindings {
			break
		}
		in := FindingInput{Kind: r.Kind, Section: cut(r.Section, 120), Text: r.Text, ICD10: r.ICD10, Drug: r.Drug, Dose: r.Dose, Frequency: r.Frequency, Duration: r.Duration}
		if _, err := ParseFindingKind(in.Kind); err != nil {
			in.Kind = "note"
		}
		in.ICD10 = CleanICD10(in.ICD10)
		in.Text = cutLines(in.Text, 4000)
		in.Drug, in.Dose, in.Frequency, in.Duration = cut(in.Drug, 200), cut(in.Dose, 200), cut(in.Frequency, 200), cut(in.Duration, 200)
		if d, err := Date(strings.TrimSpace(r.Date)); err == nil {
			in.Date = d
		}
		conf := clampConfidence(r.Confidence)
		if in.Kind == "vaccination" {
			earlier, ok := vaccines[FoldKey(in.Section)]
			if !ok || FoldKey(earlier) != FoldKey(in.Drug) {
				conf = min(conf, UncheckedVaccineConfidence)
			}
			if ok {
				in.Drug = earlier
			}
			if in.Date > today {
				product := strings.TrimSpace(CleanText(in.Drug) + " " + CleanText(in.Dose))
				in.Kind, in.Text = "advice", cutLines(strings.TrimSpace(product+"\n"+in.Text), 4000)
			}
		}
		v, err := in.Validate()
		if err != nil {
			continue
		}
		ins = append(ins, v)
		pages = append(pages, clampPage(r.Page, pageCount))
		confs = append(confs, conf)
	}
	return ins, pages, confs
}

func clampPage(p, count int) int {
	if p < 1 || p > count {
		return 1
	}
	return p
}
