package domain

import (
	"regexp"
	"slices"
	"time"
)

// MetricCategories lists the categories of catalog metrics, in catalog
// order; a suggested metric takes one of them.
var MetricCategories = []MetricCategory{CatGlucose, CatLipids, CatKidney, CatLiver, CatBloodCount, CatThyroid, CatElectrolytes,
	CatInflammation, CatIronVitamins, CatHepatitis, CatTumorMarkers, CatCoagulation, CatEnzymes, CatUrine, CatVitals}

// AnyProvider is the provider of an analyte link that applies to papers
// from every provider: a link a person approved from a suggestion
// (HW-MET-8). A link for the paper's own provider wins over it.
const AnyProvider = "*"

// customPrefix starts the code of a metric a family approved.
const customPrefix = "custom-"

// CustomMetricCode is the code of a custom metric named name.
func CustomMetricCode(name string) string { return customPrefix + slug(analyteKey(name)) }

// SuggestionKind is what the AI suggests for a printed name.
type SuggestionKind string

const (
	// SuggestLink: the name is a known metric under another name.
	SuggestLink SuggestionKind = "link"
	// SuggestNew: the name is a test the catalog lacks.
	SuggestNew SuggestionKind = "new"
	// SuggestNone: the name is not a measured test, such as a heading.
	SuggestNone SuggestionKind = "none"
)

// SuggestionStatus is where a suggestion is.
type SuggestionStatus string

const (
	SuggestionPending  SuggestionStatus = "pending"
	SuggestionApproved SuggestionStatus = "approved"
	SuggestionRejected SuggestionStatus = "rejected"
)

// MetricProposal is a suggested new metric (HW-MET-8). LOINCName is the
// code's name in the NLM LOINC service; a code the service doesn't know is
// dropped, so LOINC is "" or checked. About is the description fetched by
// that code, nil when the source has none.
type MetricProposal struct {
	Name      string         `json:"name"`
	NameVI    string         `json:"nameVi"`
	LOINC     string         `json:"loinc"`
	LOINCName string         `json:"loincName,omitempty"`
	Unit      string         `json:"unit"`
	Category  MetricCategory `json:"category"`
	About     *MetricAbout   `json:"about,omitempty"`
}

// MetricSuggestion is what the app suggests for one printed name no metric
// knows (HW-MET-8). Values counts the approved values with that name that
// are still unlinked.
type MetricSuggestion struct {
	ID         string           `json:"id"`
	AnalyteKey string           `json:"analyteKey"`
	Analyte    string           `json:"analyte"`
	Unit       string           `json:"unit"`
	Kind       SuggestionKind   `json:"kind"`
	MetricCode string           `json:"metricCode,omitempty"`
	Proposal   *MetricProposal  `json:"proposal,omitempty"`
	Reason     string           `json:"reason"`
	Model      string           `json:"model"`
	Status     SuggestionStatus `json:"status"`
	Values     int              `json:"values"`
	CreatedAt  time.Time        `json:"createdAt"`
	DecidedBy  string           `json:"decidedBy,omitempty"`
	DecidedAt  *time.Time       `json:"decidedAt,omitempty"`
}

// UnknownName is a printed name no metric knows, as the latest value
// prints it, with its unit.
type UnknownName struct {
	Key     string
	Analyte string
	Unit    string
}

// UnknownNames groups unlinked values by folded name, keeping the latest
// spelling and unit of each, in first-seen order. Values come oldest
// first; names already suggested are skipped.
func UnknownNames(obs []Observation, suggested map[string]bool) []UnknownName {
	index := map[string]int{}
	var out []UnknownName
	for _, o := range obs {
		key := analyteKey(o.Analyte)
		if key == "" || suggested[key] {
			continue
		}
		n := UnknownName{Key: key, Analyte: o.Analyte, Unit: NormalizeUnit(o.Unit)}
		if i, ok := index[key]; ok {
			out[i] = n
			continue
		}
		index[key] = len(out)
		out = append(out, n)
	}
	return out
}

// MaxSuggestNames bounds the names sent in one suggestion request.
const MaxSuggestNames = 40

// SuggestAnswer is the AI's answer for one name.
type SuggestAnswer struct {
	Name       string `json:"name"`
	Action     string `json:"action"`
	MetricCode string `json:"metric_code"`
	NewName    string `json:"new_name"`
	NewNameVI  string `json:"new_name_vi"`
	LOINC      string `json:"loinc"`
	Unit       string `json:"unit"`
	Category   string `json:"category"`
	Reason     string `json:"reason"`
}

// SuggestOutput is the AI's answer, shaped by SuggestSchema.
type SuggestOutput struct {
	Suggestions []SuggestAnswer `json:"suggestions"`
}

var loincCode = regexp.MustCompile(`^[0-9]{1,7}-[0-9]$`)

// Clean turns the answers into suggestions for the names asked about. An
// answer for a name not asked about, a second answer for a name, a link to
// a code the catalog lacks, or a new metric without a name or a valid
// category is dropped, so that name is asked again next time. A LOINC
// code that is not shaped like one is cleared. Proposals still need their
// LOINC code checked.
func (o SuggestOutput) Clean(names []UnknownName, catalog *Catalog) []MetricSuggestion {
	byName := map[string]UnknownName{}
	for _, n := range names {
		byName[n.Analyte] = n
	}
	done := map[string]bool{}
	var out []MetricSuggestion
	for _, a := range o.Suggestions {
		n, ok := byName[a.Name]
		if !ok || done[n.Key] {
			continue
		}
		s := MetricSuggestion{AnalyteKey: n.Key, Analyte: n.Analyte, Unit: n.Unit, Reason: cut(a.Reason, 300), Status: SuggestionPending}
		switch SuggestionKind(a.Action) {
		case SuggestLink:
			if _, ok := catalog.Metric(a.MetricCode); !ok {
				continue
			}
			s.Kind, s.MetricCode = SuggestLink, a.MetricCode
		case SuggestNew:
			p := MetricProposal{Name: cut(a.NewName, 80), NameVI: cut(a.NewNameVI, 80), LOINC: cut(a.LOINC, 12),
				Unit: NormalizeUnit(cut(a.Unit, 30)), Category: MetricCategory(a.Category)}
			if p.Name == "" || !slices.Contains(MetricCategories, p.Category) {
				continue
			}
			if !loincCode.MatchString(p.LOINC) {
				p.LOINC = ""
			}
			s.Kind, s.Proposal = SuggestNew, &p
		case SuggestNone:
			s.Kind, s.Status = SuggestNone, SuggestionRejected
		default:
			continue
		}
		done[n.Key] = true
		out = append(out, s)
	}
	return out
}
