package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (c *CopilotClient) getCopilotToken(ctx context.Context) (string, error) {
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
