package cache

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client talks to the Upstash Redis REST API. Commands are sent as JSON arrays
// in the request body (values never travel in the URL), invalidation uses SCAN
// instead of KEYS, and every key is namespaced with the configured prefix so
// environments never share keys (WM-OPS-2, WM-OPS-2b).
type Client struct {
	baseURL string
	token   string
	prefix  string
	client  *http.Client
	log     *logger.Logger
}

const scanBatchSize = 500

func NewClient(config domain.CacheConfig, log *logger.Logger) (*Client, error) {
	if config.RESTURL == "" || config.RESTToken == "" {
		return nil, errors.New("missing Upstash Redis REST configuration")
	}

	log.LogApplicationEvent(context.Background(), "initializing upstash cache client",
		slog.String("base_url", config.RESTURL),
		slog.String("key_prefix", config.KeyPrefix),
		slog.String("component", "upstash_cache"),
	)

	return &Client{
		baseURL: strings.TrimRight(config.RESTURL, "/"),
		token:   config.RESTToken,
		prefix:  config.KeyPrefix,
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
	args := []any{"SET", r.prefix + key, value}
	if ttlSeconds > 0 {
		args = append(args, "EX", ttlSeconds)
	}
	var result string
	return r.command(ctx, &result, args...)
}

func (r *Client) Get(ctx context.Context, key string) (string, bool, error) {
	var result *string
	if err := r.command(ctx, &result, "GET", r.prefix+key); err != nil {
		return "", false, err
	}
	if result == nil {
		return "", false, nil
	}
	return *result, true, nil
}

// Scan returns every key matching pattern (without the prefix).
func (r *Client) Scan(ctx context.Context, pattern string) ([]string, error) {
	cursor := "0"
	keys := []string{}
	for {
		var result []json.RawMessage
		if err := r.command(ctx, &result, "SCAN", cursor, "MATCH", r.prefix+pattern, "COUNT", scanBatchSize); err != nil {
			return nil, err
		}
		if len(result) != 2 {
			return nil, fmt.Errorf("unexpected SCAN reply with %d elements", len(result))
		}
		if err := json.Unmarshal(result[0], &cursor); err != nil {
			return nil, fmt.Errorf("decode SCAN cursor: %w", err)
		}
		var batch []string
		if err := json.Unmarshal(result[1], &batch); err != nil {
			return nil, fmt.Errorf("decode SCAN keys: %w", err)
		}
		for _, key := range batch {
			keys = append(keys, strings.TrimPrefix(key, r.prefix))
		}
		if cursor == "0" {
			return keys, nil
		}
	}
}

// Delete removes the given keys (without the prefix) and returns how many existed.
func (r *Client) Delete(ctx context.Context, keys ...string) (int, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	args := make([]any, 0, len(keys)+1)
	args = append(args, "DEL")
	for _, key := range keys {
		args = append(args, r.prefix+key)
	}
	var deleted int
	if err := r.command(ctx, &deleted, args...); err != nil {
		return 0, err
	}
	return deleted, nil
}

func (r *Client) Ping(ctx context.Context) error {
	var result string
	if err := r.command(ctx, &result, "PING"); err != nil {
		return err
	}
	if strings.ToUpper(result) != "PONG" {
		return fmt.Errorf("unexpected ping response: %s", result)
	}
	return nil
}

// command posts a Redis command as a JSON array and decodes its result.
func (r *Client) command(ctx context.Context, result any, args ...any) error {
	parts := make([]string, len(args))
	for i, arg := range args {
		switch typed := arg.(type) {
		case string:
			parts[i] = typed
		case int:
			parts[i] = strconv.Itoa(typed)
		default:
			parts[i] = fmt.Sprint(typed)
		}
	}
	body, err := json.Marshal(parts)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+r.token)
	request.Header.Set("Content-Type", "application/json")

	response, err := r.client.Do(request)
	if err != nil {
		return fmt.Errorf("upstash %s: %w", parts[0], err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	var payload struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("upstash %s: status %d: decode reply: %w", parts[0], response.StatusCode, err)
	}
	if payload.Error != "" {
		return fmt.Errorf("upstash %s: %s", parts[0], payload.Error)
	}
	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("upstash %s: status %d", parts[0], response.StatusCode)
	}
	if len(payload.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload.Result, result); err != nil {
		return fmt.Errorf("upstash %s: decode result: %w", parts[0], err)
	}
	return nil
}
