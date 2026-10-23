package domain

// AboutSource names where every metric quote comes from.
const AboutSource = "MedlinePlus, U.S. National Library of Medicine"

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
	a.Source = AboutSource
	a.Fetched = AboutFetched
	return a, true
}
