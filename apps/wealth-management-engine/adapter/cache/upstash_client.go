package cache

import (
	"apps/wealth-management-engine/adapter/logger"
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	client  *http.Client
	log     *logger.Logger
}

func NewClient(config domain.CacheConfig, log *logger.Logger) (*Client, error) {
	if config.RESTURL == "" || config.RESTToken == "" {
		return nil, errors.New("missing Upstash Redis REST configuration")
	}

	log.LogApplicationEvent(context.Background(), "initializing upstash cache client",
		slog.String("base_url", config.RESTURL),
		slog.String("component", "upstash_cache"),
	)

	return &Client{
		baseURL: strings.TrimRight(config.RESTURL, "/"),
		token:   config.RESTToken,
		client:  &http.Client{Timeout: 10 * time.Second},
		log:     log,
	}, nil
}

func NewClientWithHTTPClient(config domain.CacheConfig, client *http.Client, log *logger.Logger) (*Client, error) {
	cacheClient, err := NewClient(config, log)
	if err != nil {
		return nil, err
	}
	cacheClient.client = client
	return cacheClient, nil
}

func (r *Client) Set(ctx context.Context, key string, value string, ttlSeconds int) error {
	path := fmt.Sprintf("/set/%s/%s", url.PathEscape(key), url.PathEscape(value))
	if ttlSeconds > 0 {
		path = fmt.Sprintf("%s?EX=%d", path, ttlSeconds)
	}

	response, err := r.do(ctx, http.MethodPost, path)
	if err != nil {
		return err
	}

	var payload responsePayload[string]
	if err := json.Unmarshal(response, &payload); err != nil {
		return err
	}

	if payload.Error != "" {
		return errors.New(payload.Error)
	}

	return nil
}

func (r *Client) Get(ctx context.Context, key string) (string, bool, error) {
	response, err := r.do(ctx, http.MethodGet, fmt.Sprintf("/get/%s", url.PathEscape(key)))
	if err != nil {
		return "", false, err
	}

	var payload responsePayload[*string]
	if err := json.Unmarshal(response, &payload); err != nil {
		return "", false, err
	}
	if payload.Error != "" {
		return "", false, errors.New(payload.Error)
	}
	if payload.Result == nil {
		return "", false, nil
	}

	return *payload.Result, true, nil
}

func (r *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	response, err := r.do(ctx, http.MethodGet, fmt.Sprintf("/keys/%s", url.PathEscape(pattern)))
	if err != nil {
		return nil, err
	}

	var payload responsePayload[[]string]
	if err := json.Unmarshal(response, &payload); err != nil {
		return nil, err
	}
	if payload.Error != "" {
		return nil, errors.New(payload.Error)
	}

	return payload.Result, nil
}

func (r *Client) Delete(ctx context.Context, key string) error {
	response, err := r.do(ctx, http.MethodPost, fmt.Sprintf("/del/%s", url.PathEscape(key)))
	if err != nil {
		return err
	}

	var payload responsePayload[int]
	if err := json.Unmarshal(response, &payload); err != nil {
		return err
	}
	if payload.Error != "" {
		return errors.New(payload.Error)
	}

	return nil
}

func (r *Client) Ping(ctx context.Context) error {
	response, err := r.do(ctx, http.MethodGet, "/ping")
	if err != nil {
		return err
	}

	var payload responsePayload[string]
	if err := json.Unmarshal(response, &payload); err != nil {
		return err
	}
	if payload.Error != "" {
		return errors.New(payload.Error)
	}
	if strings.ToUpper(payload.Result) != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", payload.Result)
	}

	return nil
}

func (r *Client) do(ctx context.Context, method string, path string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, method, r.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.token))

	response, err := r.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("upstash request failed with status %d: %s", response.StatusCode, string(body))
	}

	return body, nil
}

type responsePayload[T any] struct {
	Result T      `json:"result"`
	Error  string `json:"error"`
}
