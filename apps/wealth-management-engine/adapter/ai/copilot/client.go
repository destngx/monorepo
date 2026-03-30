package copilot

import (
	"apps/wealth-management-engine/domain"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	config     domain.AIConfig
	httpClient *http.Client

	mu           sync.Mutex
	cachedToken  string
	cachedExpiry int64
}

func NewClient(config domain.AIConfig) (*Client, error) {
	if config.GitHubToken == "" {
		return nil, errors.New("missing GitHub token")
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func NewClientWithHTTPClient(config domain.AIConfig, httpClient *http.Client) (*Client, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	client.httpClient = httpClient
	return client, nil
}

func (c *Client) StreamCompletion(ctx context.Context, prompt string, model string) (io.ReadCloser, error) {
	token, err := c.getCopilotToken(ctx)
	if err != nil {
		return nil, err
	}

	requestBody := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"stream": true,
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(c.config.CopilotAPIBase, "/")+"/chat/completions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Editor-Version", "vscode/1.80.0")
	request.Header.Set("Editor-Plugin-Version", "copilot-chat/0.1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		defer response.Body.Close()
		body, _ := io.ReadAll(response.Body)
		return nil, fmt.Errorf("copilot completion failed with status %d: %s", response.StatusCode, string(body))
	}

	return response.Body, nil
}

func (c *Client) CompleteJSON(ctx context.Context, messages []domain.RoleMessage, model string) (string, error) {
	token, err := c.getCopilotToken(ctx)
	if err != nil {
		return "", err
	}

	requestBody := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   false,
		"response_format": map[string]string{
			"type": "json_object",
		},
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(c.config.CopilotAPIBase, "/")+"/chat/completions",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Editor-Version", "vscode/1.80.0")
	request.Header.Set("Editor-Plugin-Version", "copilot-chat/0.1.0")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("copilot completion failed with status %d: %s", response.StatusCode, string(body))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(response.Body).Decode(&completion); err != nil {
		return "", err
	}
	if len(completion.Choices) == 0 {
		return "", errors.New("copilot completion returned no choices")
	}

	return completion.Choices[0].Message.Content, nil
}

func (c *Client) getCopilotToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	if c.cachedToken != "" && c.cachedExpiry > time.Now().Unix()+300 {
		token := c.cachedToken
		c.mu.Unlock()
		return token, nil
	}
	c.mu.Unlock()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		strings.TrimRight(c.config.GitHubAPIBase, "/")+"/copilot_internal/v2/token",
		nil,
	)
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "token "+c.config.GitHubToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		return "", fmt.Errorf("copilot token exchange failed with status %d: %s", response.StatusCode, string(body))
	}

	var tokenPayload struct {
		Token     string `json:"token"`
		ExpiresAt int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(response.Body).Decode(&tokenPayload); err != nil {
		return "", err
	}
	if tokenPayload.Token == "" {
		return "", errors.New("copilot token exchange returned empty token")
	}

	c.mu.Lock()
	c.cachedToken = tokenPayload.Token
	c.cachedExpiry = tokenPayload.ExpiresAt
	c.mu.Unlock()

	return tokenPayload.Token, nil
}
