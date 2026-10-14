// Package exa is the second web search provider (WM-AI-12).
package exa

import (
	"apps/wealth-management/adapter/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"net/http"
	"strings"
)

const DefaultBaseURL = "https://api.exa.ai"

// snippetCharacters bounds the page text Exa returns per result.
const snippetCharacters = 1500

type Provider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

var _ port.SearchProvider = (*Provider)(nil)

// NewProvider needs an API key; it is sent in x-api-key, never logged.
func NewProvider(baseURL string, apiKey string) (*Provider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("missing EXA_API_KEY")
	}
	return &Provider{baseURL: strings.TrimRight(baseURL, "/"), apiKey: strings.TrimSpace(apiKey), httpClient: httpjson.NewClient()}, nil
}

func (p *Provider) Name() string { return "exa" }

func (p *Provider) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	var payload struct {
		Results []struct {
			Title      string   `json:"title"`
			URL        string   `json:"url"`
			Text       string   `json:"text"`
			Summary    string   `json:"summary"`
			Highlights []string `json:"highlights"`
		} `json:"results"`
	}
	body := map[string]any{
		"query":      query,
		"type":       "auto",
		"numResults": limit,
		"contents":   map[string]any{"text": map[string]any{"maxCharacters": snippetCharacters}},
	}
	if err := httpjson.Post(ctx, p.httpClient, p.Name(), p.baseURL+"/search", map[string]string{"x-api-key": p.apiKey}, body, &payload); err != nil {
		return nil, err
	}
	results := make([]domain.SearchResult, 0, len(payload.Results))
	for _, row := range payload.Results {
		snippet := row.Text
		if snippet == "" {
			snippet = row.Summary
		}
		if snippet == "" && len(row.Highlights) > 0 {
			snippet = row.Highlights[0]
		}
		if result, ok := domain.NewSearchResult(row.Title, row.URL, snippet); ok {
			results = append(results, result)
		}
	}
	return results, nil
}
