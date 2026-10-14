// Package marketdata is the engine's only market data adapter (WM-MKT-1):
// it calls apps/market-data-server, which owns every source (vnstock,
// fmarket, Yahoo Finance, CoinGecko, currency-api), the provider chains,
// caching and last-good copies. See docs/market-data-server/MARKET_PLATFORM.md.
package marketdata

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is where the market data server listens by default.
const DefaultBaseURL = "http://localhost:8000"

// Timeout covers a full provider chain on the server: each source call is
// bounded at 10s and a chain has at most three sources.
const Timeout = 35 * time.Second

// maxErrorBody bounds how much of an error answer is read.
const maxErrorBody = 64 << 10

// Client calls the market data server's /api/v2.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient returns a client for the server at baseURL.
func NewClient(baseURL string) (*Client, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("%w: MARKET_DATA_SERVER_URL must be an absolute URL, got %q", domain.ErrValidation, baseURL)
	}
	return &Client{baseURL: trimmed, httpClient: &http.Client{Timeout: Timeout}}, nil
}

// BaseURL is the server address, for logs.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// meta is the server's data-source indicator for one answer.
type meta struct {
	Provider string   `json:"provider"`
	Fallback []string `json:"fallback"`
	Cached   bool     `json:"cached"`
	Stale    bool     `json:"stale"`
	AsOf     string   `json:"as_of"`
}

type envelope[T any] struct {
	Data T    `json:"data"`
	Meta meta `json:"meta"`
}

// ServerError is an error answer from the server. Status 400 wraps
// domain.ErrValidation and 404 domain.ErrNotFound, so handlers map them to
// client errors; anything else is an upstream failure.
type ServerError struct {
	Status  int
	Code    string
	Message string
}

func (e *ServerError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("market data server answered %d", e.Status)
	}
	return "market data server: " + e.Message
}

func (e *ServerError) Unwrap() error {
	switch e.Status {
	case http.StatusBadRequest:
		return domain.ErrValidation
	case http.StatusNotFound:
		return domain.ErrNotFound
	default:
		return nil
	}
}

// get requests path and decodes the `data` of the answer into out. A stale
// answer (every source failed and the server served its last-good copy) is
// recorded on the request with domain.MarkStale, so the engine sets its own
// X-Data-Stale header (WM-NFR-4). ShouldBypassCache becomes skip_cache.
func get[T any](ctx context.Context, c *Client, path string, query url.Values) (T, meta, error) {
	var answer envelope[T]
	if query == nil {
		query = url.Values{}
	}
	if domain.ShouldBypassCache(ctx) {
		query.Set("skip_cache", "true")
	}
	target := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return answer.Data, answer.Meta, fmt.Errorf("market data server: build request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return answer.Data, answer.Meta, fmt.Errorf("market data server: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return answer.Data, answer.Meta, readError(response)
	}
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		return answer.Data, answer.Meta, fmt.Errorf("market data server: decode %s: %w", path, err)
	}
	if answer.Meta.Stale {
		asOf, err := time.Parse(time.RFC3339, answer.Meta.AsOf)
		if err != nil {
			// Still stale, just without a known age.
			asOf = time.Unix(0, 0).UTC()
		}
		domain.MarkStale(ctx, asOf)
	}
	return answer.Data, answer.Meta, nil
}

func readError(response *http.Response) error {
	serverErr := &ServerError{Status: response.StatusCode}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
	if err != nil {
		return errors.Join(serverErr, fmt.Errorf("read error answer: %w", err))
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Detail any `json:"detail"`
	}
	if json.Unmarshal(raw, &body) == nil {
		serverErr.Code, serverErr.Message = body.Error.Code, body.Error.Message
		if serverErr.Message == "" && body.Detail != nil {
			serverErr.Message = fmt.Sprint(body.Detail)
		}
	}
	return serverErr
}
