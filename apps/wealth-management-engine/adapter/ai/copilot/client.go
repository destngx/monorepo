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
	if config.GitHubToken == "" && config.CopilotBearerToken == "" {
		return nil, errors.New("missing GitHub token or Copilot bearer token")
	}
	if strings.TrimSpace(config.EditorVersion) == "" {
		config.EditorVersion = "vscode/1.80.0"
	}
	if strings.TrimSpace(config.EditorPluginVersion) == "" {
		config.EditorPluginVersion = "copilot-chat/0.1.0"
	}
	if strings.TrimSpace(config.CopilotIntegrationID) == "" {
		config.CopilotIntegrationID = "vscode-chat"
	}
	if strings.TrimSpace(config.UserAgent) == "" {
		config.UserAgent = "GitHubCopilotChat/0.1.0"
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
	c.applyCopilotHeaders(request)

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
	c.applyCopilotHeaders(request)

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
	if token := strings.TrimSpace(c.config.CopilotBearerToken); token != "" {
		return token, nil
	}

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
	c.applyCopilotHeaders(request)

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

func (c *Client) applyCopilotHeaders(request *http.Request) {
	request.Header.Set("Editor-Version", c.config.EditorVersion)
	request.Header.Set("Editor-Plugin-Version", c.config.EditorPluginVersion)
	request.Header.Set("Copilot-Integration-Id", c.config.CopilotIntegrationID)
	request.Header.Set("User-Agent", c.config.UserAgent)
}
