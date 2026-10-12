package domain

import (
	"net/url"
	"strings"
)

// SearchTTLSeconds keeps web search results for 14 days (WM-AI-12).
const SearchTTLSeconds = 14 * 24 * 3600

// SearchResultLimit is how many results one search returns.
const SearchResultLimit = 5

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	// Source is the publisher host without "www.".
	Source string `json:"source"`
}

// SearchResponse is the answer of the first provider that returned results.
type SearchResponse struct {
	Query    string         `json:"query"`
	Provider string         `json:"provider"`
	Results  []SearchResult `json:"results"`
	// Fallback lists the providers tried before Provider and why they failed.
	Fallback string `json:"fallback,omitempty"`
}

// NormalizeSearchQuery collapses whitespace and case so equivalent queries
// share a cache entry.
func NormalizeSearchQuery(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}

// SearchSource is the host of a result URL without "www.", or "" when the
// URL does not parse.
func SearchSource(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
}

// NewSearchResult trims the fields and fills Source; ok is false when the
// result has no usable URL.
func NewSearchResult(title string, link string, snippet string) (SearchResult, bool) {
	link = strings.TrimSpace(link)
	source := SearchSource(link)
	if source == "" || !strings.HasPrefix(link, "http") {
		return SearchResult{}, false
	}
	return SearchResult{
		Title:   strings.Join(strings.Fields(title), " "),
		URL:     link,
		Snippet: strings.Join(strings.Fields(snippet), " "),
		Source:  source,
	}, true
}

// SearchConfig holds the optional paid search keys; DuckDuckGo needs none
// and is always the last provider (WM-AI-12).
type SearchConfig struct {
	TavilyAPIKey string
	ExaAPIKey    string
}
