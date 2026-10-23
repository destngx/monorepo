package domain

import (
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// PersonObservation is an approved observation of a person with the visit
// and place it came from, and its page: the page's number within the
// document, counting across its files, and title. A typed-in measurement
// has none of them.
type PersonObservation struct {
	Observation
	VisitID    string
	Provider   string
	PageNumber int
	PageTitle  string
}

// PointSource says where a trend point came from.
type PointSource string

const (
	SourcePaper    PointSource = "paper"
	SourceTyped    PointSource = "typed"
	SourceComputed PointSource = "computed"
)

// TrendPoint is one value on a person's trend chart (HW-MET-3). Value and
// the range are in the metric's canonical unit; Value is nil for a text
// result or a unit with no known conversion, which a table still lists.
// The range and flag are the paper's own and never made up (HW-MET-5).
type TrendPoint struct {
	ObservationID string      `json:"observationId,omitempty"`
	DocumentID    string      `json:"documentId,omitempty"`
	PageID        string      `json:"pageId,omitempty"`
	PageNumber    int         `json:"pageNumber,omitempty"`
	PageTitle     string      `json:"pageTitle,omitempty"`
	VisitID       string      `json:"visitId,omitempty"`
	Provider      string      `json:"provider,omitempty"`
	Date          string      `json:"date"`
	Value         *float64    `json:"value,omitempty"`
	Printed       string      `json:"printed"`
	PrintedUnit   string      `json:"printedUnit,omitempty"`
	RefLow        *float64    `json:"refLow,omitempty"`
	RefHigh       *float64    `json:"refHigh,omitempty"`
	RefText       string      `json:"refText,omitempty"`
	Flag          string      `json:"flag,omitempty"`
	Source        PointSource `json:"source"`
}

// TrendPointOf turns an observation into a point of metric m.
func TrendPointOf(m Metric, o PersonObservation) TrendPoint {
	p := TrendPoint{ObservationID: o.ID, DocumentID: o.DocumentID, PageID: o.PageID, PageNumber: o.PageNumber, PageTitle: o.PageTitle, VisitID: o.VisitID, Provider: o.Provider, Date: o.SampleDate,
		Printed: o.Value, PrintedUnit: o.Unit, RefText: o.RefText, Flag: o.Flag, Source: SourceTyped}
	if o.DocumentID != "" {
		p.Source = SourcePaper
	}
	if m.Unit == "" && !m.Printed {
		return p
	}
	canonical := func(v *float64) *float64 {
		if v == nil {
			return nil
		}
		c, ok := m.ToCanonical(*v, o.Unit)
		if !ok {
			return nil
		}
		return &c
	}
	p.Value, p.RefLow, p.RefHigh = canonical(o.ValueNum), canonical(o.RefLow), canonical(o.RefHigh)
	if p.Value == nil {
		// A range in a unit the chart can't draw would sit on the wrong scale.
		p.RefLow, p.RefHigh = nil, nil
	}
	return p
}

// CatPrinted groups the metrics made from names the catalog doesn't know.
const CatPrinted MetricCategory = "printed"

// printedPrefix starts the code of a metric made from a printed name.
const printedPrefix = "printed-"

// PrintedMetricCode is the code of the metric an analyte name the catalog
// doesn't know becomes (HW-MET-7): its folded name, so `Tỷ lệ % bạch cầu
// Mid` and `TY LE % BACH CAU MID` are one metric, and `Mid%` and `Mid#`
// stay apart.
func PrintedMetricCode(analyte string) string {
	var b strings.Builder
	dash := false
	for _, r := range analyteKey(analyte) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.TrimSuffix(b.String(), "-")
	if slug == "" {
		slug = "unnamed"
	}
	return printedPrefix + slug
}

// PrintedMetric is the metric of values whose name the catalog doesn't
// know, oldest first (HW-MET-7). It is named as the latest paper prints
// it, and its unit is the latest numeric value's, so values in that unit
// are charted and others are only listed.
func PrintedMetric(code string, obs []PersonObservation) Metric {
	m := Metric{Code: code, Category: CatPrinted, Aliases: []string{}, Printed: true}
	if len(obs) > 0 {
		m.Name = obs[len(obs)-1].Analyte
	}
	for i := len(obs) - 1; i >= 0; i-- {
		if obs[i].ValueNum != nil {
			m.Unit = NormalizeUnit(obs[i].Unit)
			break
		}
	}
	return m
}

// SortPoints orders points by date, oldest first, keeping the given order
// within a day.
func SortPoints(points []TrendPoint) {
	sort.SliceStable(points, func(i, j int) bool { return points[i].Date < points[j].Date })
}

// childBMIHeightDays is how old a height may be when a weight is taken
// before 18: children grow, so an old height would give a wrong BMI.
const childBMIHeightDays = 90

// ComputeBMI derives BMI points from weights (kg) and heights (cm): each
// weight uses the latest height taken on or before its day (HW-MET-2). A
// day that already has a BMI in printed gets none, and a computed BMI has
// no range or flag, because the app never makes one up (HW-MET-5).
func ComputeBMI(weights, heights, printed []TrendPoint, birthDate string) []TrendPoint {
	hs := make([]TrendPoint, 0, len(heights))
	for _, h := range heights {
		if h.Value != nil && *h.Value > 0 && h.Date != "" {
			hs = append(hs, h)
		}
	}
	SortPoints(hs)
	taken := map[string]bool{}
	for _, p := range printed {
		taken[p.Date] = true
	}
	out := []TrendPoint{}
	for _, w := range weights {
		if w.Value == nil || *w.Value <= 0 || w.Date == "" || taken[w.Date] {
			continue
		}
		i := sort.Search(len(hs), func(i int) bool { return hs[i].Date > w.Date }) - 1
		if i < 0 || (isMinorOn(birthDate, w.Date) && daysBetween(hs[i].Date, w.Date) > childBMIHeightDays) {
			continue
		}
		m := *hs[i].Value / 100
		bmi := math.Round(*w.Value/(m*m)*10) / 10
		taken[w.Date] = true
		out = append(out, TrendPoint{Date: w.Date, Value: &bmi, Printed: formatOneDecimal(bmi), PrintedUnit: "kg/m²",
			Source: SourceComputed})
	}
	return out
}

func formatOneDecimal(v float64) string {
	return strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
}

// isMinorOn reports whether someone born on birth is under 18 on day; an
// unknown birth date counts as an adult.
func isMinorOn(birth, day string) bool {
	b, err1 := time.Parse(DateLayout, birth)
	d, err2 := time.Parse(DateLayout, day)
	if err1 != nil || err2 != nil {
		return false
	}
	return d.Before(b.AddDate(18, 0, 0))
}

func daysBetween(from, to string) int {
	f, err1 := time.Parse(DateLayout, from)
	t, err2 := time.Parse(DateLayout, to)
	if err1 != nil || err2 != nil {
		return math.MaxInt
	}
	return int(t.Sub(f).Hours() / 24)
}

// OverviewItem is one metric on a person's overview (HW-MET-4): the latest
// value with its flag, the change since the previous value when both are
// numbers, and the recent numeric values for a sparkline.
type OverviewItem struct {
	Metric   Metric       `json:"metric"`
	Latest   TrendPoint   `json:"latest"`
	Previous *TrendPoint  `json:"previous,omitempty"`
	Change   *float64     `json:"change,omitempty"`
	Count    int          `json:"count"`
	Recent   []TrendPoint `json:"recent"`
}

// OverviewGroup is the overview's metrics of one category.
type OverviewGroup struct {
	Category MetricCategory `json:"category"`
	Items    []OverviewItem `json:"items"`
}

// sparkPoints is how many recent values an overview item carries.
const sparkPoints = 12

// OverviewOf builds an overview item from a metric's points, sorted oldest
// first; it is false when there are none.
func OverviewOf(m Metric, points []TrendPoint) (OverviewItem, bool) {
	if len(points) == 0 {
		return OverviewItem{}, false
	}
	item := OverviewItem{Metric: m, Latest: points[len(points)-1], Count: len(points), Recent: []TrendPoint{}}
	if len(points) > 1 {
		prev := points[len(points)-2]
		item.Previous = &prev
		if item.Latest.Value != nil && prev.Value != nil {
			d := math.Round((*item.Latest.Value-*prev.Value)*1e6) / 1e6
			item.Change = &d
		}
	}
	for i := len(points) - 1; i >= 0 && len(item.Recent) < sparkPoints; i-- {
		if points[i].Value != nil {
			item.Recent = append(item.Recent, points[i])
		}
	}
	for i, j := 0, len(item.Recent)-1; i < j; i, j = i+1, j-1 {
		item.Recent[i], item.Recent[j] = item.Recent[j], item.Recent[i]
	}
	return item, true
}

// MeasurementInput is a set of vitals typed in for one day without a
// paper, such as a home blood pressure reading (HW-MET-2).
type MeasurementInput struct {
	Date   string             `json:"date"`
	Values []MeasurementValue `json:"values"`
}

// MeasurementValue is one typed vital; an empty unit is the metric's own.
type MeasurementValue struct {
	MetricCode string `json:"metricCode"`
	Value      string `json:"value"`
	Unit       string `json:"unit"`
}

// maxMeasurements bounds one typed-in set.
const maxMeasurements = 20

// Validate checks the day and that each value is a positive number of a
// vitals metric in a unit the metric converts from. Empty values are dropped.
func (in MeasurementInput) Validate(today string, catalog *Catalog) (MeasurementInput, error) {
	day, err := Date(in.Date)
	if err != nil {
		return in, Invalid("date must be YYYY-MM-DD")
	}
	if day > today {
		return in, Invalid("date is in the future")
	}
	out := MeasurementInput{Date: day, Values: []MeasurementValue{}}
	seen := map[string]bool{}
	for _, v := range in.Values {
		v.Value, v.Unit = CleanText(v.Value), NormalizeUnit(CleanText(v.Unit))
		if v.Value == "" {
			continue
		}
		m, ok := catalog.Metric(v.MetricCode)
		if !ok || m.Category != CatVitals {
			return in, Invalid("%q is not a vital sign or body measurement", v.MetricCode)
		}
		if seen[m.Code] {
			return in, Invalid("%s is given twice", m.Name)
		}
		seen[m.Code] = true
		if v.Unit == "" {
			v.Unit = m.Unit
		}
		n := ParseNumber(v.Value)
		if n == nil || *n <= 0 {
			return in, Invalid("%s must be a positive number", m.Name)
		}
		if _, ok := m.ToCanonical(*n, v.Unit); !ok {
			return in, Invalid("%s can't be given in %s", m.Name, v.Unit)
		}
		out.Values = append(out.Values, v)
	}
	if len(out.Values) == 0 {
		return in, Invalid("type at least one value")
	}
	if len(out.Values) > maxMeasurements {
		return in, Invalid("at most %d values at once", maxMeasurements)
	}
	return out, nil
}
