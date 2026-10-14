package domain

// DataSource says where a market answer came from (WM-INV-15), so the UI can
// show it next to the figure: the provider that answered, the providers
// that failed before it, and whether the answer is the market data server's
// last-good copy because every provider failed.
type DataSource struct {
	Provider string `json:"provider"`
	// Fallback lists the providers tried first, as "name: reason".
	Fallback []string `json:"fallback,omitempty"`
	// Stale means every provider failed and AsOf is the age of the copy.
	Stale bool `json:"stale,omitempty"`
	// AsOf is when the data was read from the provider (RFC 3339 UTC).
	AsOf string `json:"asOf,omitempty"`
}
