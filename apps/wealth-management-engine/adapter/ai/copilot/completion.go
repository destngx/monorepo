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
)

func (c *CopilotClient) StreamCompletion(ctx context.Context, prompt string, model string) (io.ReadCloser, error) {
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

func (c *CopilotClient) CompleteJSON(ctx context.Context, messages []domain.RoleMessage, model string) (string, error) {
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
