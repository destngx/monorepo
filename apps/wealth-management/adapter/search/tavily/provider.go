// Package tavily is the primary web search provider (WM-AI-12).
package tavily

import (
	"apps/wealth-management/adapter/market/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"net/http"
	"strings"
)

const DefaultBaseURL = "https://api.tavily.com"

type Provider struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

var _ port.SearchProvider = (*Provider)(nil)

// NewProvider needs an API key; it is sent as a bearer token, never logged.
func NewProvider(baseURL string, apiKey string) (*Provider, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("missing TAVILY_API_KEY")
	}
	return &Provider{baseURL: strings.TrimRight(baseURL, "/"), apiKey: strings.TrimSpace(apiKey), httpClient: httpjson.NewClient()}, nil
}

func (p *Provider) Name() string { return "tavily" }

func (p *Provider) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	var payload struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	body := map[string]any{"query": query, "search_depth": "advanced", "max_results": limit}
	headers := map[string]string{"Authorization": "Bearer " + p.apiKey}
	if err := httpjson.Post(ctx, p.httpClient, p.Name(), p.baseURL+"/search", headers, body, &payload); err != nil {
		return nil, err
	}
	results := make([]domain.SearchResult, 0, len(payload.Results))
	for _, row := range payload.Results {
		if result, ok := domain.NewSearchResult(row.Title, row.URL, row.Content); ok {
			results = append(results, result)
		}
	}
	return results, nil
}
