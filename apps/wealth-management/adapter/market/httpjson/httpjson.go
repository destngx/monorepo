// Package httpjson is the shared request-and-decode helper for adapters
// that call public JSON APIs.
package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultTimeout bounds one upstream call so a slow provider fails over fast.
const DefaultTimeout = 10 * time.Second

// NewClient returns an HTTP client with DefaultTimeout.
func NewClient() *http.Client {
	return &http.Client{Timeout: DefaultTimeout}
}

// StatusError is a non-2xx answer; the body is not kept because upstream
// error pages are large and may echo request details.
type StatusError struct {
	Provider string
	Status   int
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s request failed with status %d", e.Provider, e.Status)
}

// Get requests url with headers and decodes the JSON body into out.
func Get(ctx context.Context, client *http.Client, provider string, url string, headers map[string]string, out any) error {
	return do(ctx, client, provider, http.MethodGet, url, headers, nil, out)
}

// Post sends body as JSON and decodes the JSON answer into out.
func Post(ctx context.Context, client *http.Client, provider string, url string, headers map[string]string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("%s: encode request: %w", provider, err)
	}
	return do(ctx, client, provider, http.MethodPost, url, headers, payload, out)
}

func do(ctx context.Context, client *http.Client, provider string, method string, url string, headers map[string]string, payload []byte, out any) error {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return fmt.Errorf("%s: build request: %w", provider, err)
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%s: %w", provider, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		return &StatusError{Provider: provider, Status: response.StatusCode}
	}
	if err := json.NewDecoder(response.Body).Decode(out); err != nil {
		return fmt.Errorf("%s: decode response: %w", provider, err)
	}
	return nil
}
