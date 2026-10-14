// Package duckduckgo is the keyless last-resort web search provider
// (WM-AI-12). It reads the HTML endpoint, which has no official API, so a
// layout change or a bot challenge shows up as "no results".
package duckduckgo

import (
	"apps/wealth-management/adapter/httpjson"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/html"
)

const DefaultBaseURL = "https://html.duckduckgo.com"

// maxPageBytes bounds the HTML read from one results page.
const maxPageBytes = 2 << 20

type Provider struct {
	baseURL    string
	httpClient *http.Client
}

var _ port.SearchProvider = (*Provider)(nil)

func NewProvider(baseURL string) *Provider {
	return &Provider{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpjson.NewClient()}
}

func (p *Provider) Name() string { return "duckduckgo" }

func (p *Provider) Search(ctx context.Context, query string, limit int) ([]domain.SearchResult, error) {
	form := url.Values{"q": {query}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/html/", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: build request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "text/html")
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; wealth-management/1.0)")
	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return nil, &httpjson.StatusError{Provider: p.Name(), Status: response.StatusCode}
	}
	document, err := html.Parse(io.LimitReader(response.Body, maxPageBytes))
	if err != nil {
		return nil, fmt.Errorf("duckduckgo: parse page: %w", err)
	}
	return parseResults(document, limit), nil
}

// parseResults reads organic results (div.result, not .result--ad): the
// title link a.result__a and the snippet .result__snippet.
func parseResults(document *html.Node, limit int) []domain.SearchResult {
	var results []domain.SearchResult
	var walk func(node *html.Node)
	walk = func(node *html.Node) {
		if len(results) >= limit {
			return
		}
		if node.Type == html.ElementNode && hasClass(node, "result") {
			if !hasClass(node, "result--ad") {
				if result, ok := parseResult(node); ok {
					results = append(results, result)
				}
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)
	return results
}

func parseResult(node *html.Node) (domain.SearchResult, bool) {
	link := find(node, "result__a")
	if link == nil {
		return domain.SearchResult{}, false
	}
	snippet := ""
	if element := find(node, "result__snippet"); element != nil {
		snippet = text(element)
	}
	return domain.NewSearchResult(text(link), resolveLink(attribute(link, "href")), snippet)
}

// resolveLink unwraps DuckDuckGo's redirect (//duckduckgo.com/l/?uddg=...).
func resolveLink(href string) string {
	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if target := parsed.Query().Get("uddg"); target != "" {
		return target
	}
	if parsed.Scheme == "" {
		return ""
	}
	return href
}

func find(node *html.Node, class string) *html.Node {
	if node.Type == html.ElementNode && hasClass(node, class) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := find(child, class); found != nil {
			return found
		}
	}
	return nil
}

func hasClass(node *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attribute(node, "class")), class)
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func text(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}
