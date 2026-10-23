package domain

// The sources metric quotes come from. MedlinePlus is the default; a test
// MedlinePlus never defines, such as granulocytes, is quoted from the NCI
// Dictionary of Cancer Terms.
const (
	AboutSource    = "MedlinePlus, U.S. National Library of Medicine"
	AboutSourceNCI = "NCI Dictionary of Cancer Terms, National Cancer Institute"
)

// MetricAbout is what a metric is, quoted word for word from a trusted source (HW-MET-6).
type MetricAbout struct {
	Quote   string `json:"quote"`
	Source  string `json:"source"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Fetched string `json:"fetched"`
}

// AboutMetric returns the quote for a catalog code; false when the source
// has no page that says what the metric is.
func AboutMetric(code string) (MetricAbout, bool) {
	a, ok := metricAbout[code]
	if !ok {
		return MetricAbout{}, false
	}
	if a.Source == "" {
		a.Source = AboutSource
	}
	a.Fetched = AboutFetched
	return a, true
}
