// Package openai is an AIClient for any OpenAI chat completions compatible
// endpoint: the monorepo ai-gateway, OpenAI, or another provider (D10).
package openai

import (
	"apps/wealth-management/domain"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const (
	providerHeader = "X-AI-Provider"
	maxErrorBody   = 2048
	maxSSELine     = 4 << 20
)

type Client struct {
	baseURL  string
	apiKey   string
	provider string
	http     *http.Client
}

// NewClient talks to baseURL (".../v1"). Request deadlines come from the
// caller's context so long streams are not cut by a client-wide timeout.
func NewClient(config domain.AIConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL:  strings.TrimRight(config.BaseURL, "/"),
		apiKey:   config.APIKey,
		provider: config.Provider,
		http:     httpClient,
	}
}

func (c *Client) post(ctx context.Context, body wireRequest) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode ai request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	request.Header.Set("Content-Type", "application/json")
	if body.Stream {
		request.Header.Set("Accept", "text/event-stream")
	}
	c.authorize(request)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	if response.StatusCode/100 != 2 {
		defer response.Body.Close()
		return nil, providerError(response)
	}
	return response, nil
}

func (c *Client) authorize(request *http.Request) {
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.provider != "" {
		request.Header.Set(providerHeader, c.provider)
	}
}

func providerError(response *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
	message := strings.TrimSpace(string(raw))
	var decoded wireResponse
	if json.Unmarshal(raw, &decoded) == nil && decoded.Error != nil && decoded.Error.Message != "" {
		message = decoded.Error.Message
	}
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("%w: provider returned %d: %s", domain.ErrAIUnavailable, response.StatusCode, message)
}

func (c *Client) Complete(ctx context.Context, request domain.AIRequest) (domain.AIResponse, error) {
	response, err := c.post(ctx, encodeRequest(request, false))
	if err != nil {
		return domain.AIResponse{}, err
	}
	defer response.Body.Close()
	var decoded wireResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return domain.AIResponse{}, fmt.Errorf("%w: decode provider response: %v", domain.ErrAIUnavailable, err)
	}
	if decoded.Error != nil {
		return domain.AIResponse{}, fmt.Errorf("%w: %s", domain.ErrAIUnavailable, decoded.Error.Message)
	}
	if len(decoded.Choices) == 0 || decoded.Choices[0].Message == nil {
		return domain.AIResponse{}, fmt.Errorf("%w: provider returned no choices", domain.ErrAIUnavailable)
	}
	choice := decoded.Choices[0]
	result := domain.AIResponse{Model: firstNonEmpty(decoded.Model, request.Model), FinishReason: deref(choice.FinishReason)}
	result.Content = deref(choice.Message.Content)
	for _, call := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, domain.AIToolCall{ID: call.ID, Name: call.Function.Name, Arguments: deref(call.Function.Arguments)})
	}
	if decoded.Usage != nil {
		result.Usage = domain.AIUsage{PromptTokens: decoded.Usage.PromptTokens, CompletionTokens: decoded.Usage.CompletionTokens}
	}
	return result, nil
}

func (c *Client) Stream(ctx context.Context, request domain.AIRequest, onDelta func(string) error) (domain.AIResponse, error) {
	response, err := c.post(ctx, encodeRequest(request, true))
	if err != nil {
		return domain.AIResponse{}, err
	}
	defer response.Body.Close()

	result := domain.AIResponse{Model: request.Model}
	var content strings.Builder
	calls := map[int]*domain.AIToolCall{}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), maxSSELine)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk wireResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return domain.AIResponse{}, fmt.Errorf("%w: decode stream chunk: %v", domain.ErrAIUnavailable, err)
		}
		if chunk.Error != nil {
			return domain.AIResponse{}, fmt.Errorf("%w: %s", domain.ErrAIUnavailable, chunk.Error.Message)
		}
		if chunk.Model != "" {
			result.Model = chunk.Model
		}
		if chunk.Usage != nil {
			result.Usage = domain.AIUsage{PromptTokens: chunk.Usage.PromptTokens, CompletionTokens: chunk.Usage.CompletionTokens}
		}
		for _, choice := range chunk.Choices {
			if choice.FinishReason != nil {
				result.FinishReason = *choice.FinishReason
			}
			if choice.Delta == nil {
				continue
			}
			if delta := deref(choice.Delta.Content); delta != "" {
				content.WriteString(delta)
				if err := onDelta(delta); err != nil {
					return domain.AIResponse{}, err
				}
			}
			mergeToolCallDeltas(calls, choice.Delta.ToolCalls)
		}
	}
	if err := scanner.Err(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.AIResponse{}, ctxErr
		}
		return domain.AIResponse{}, fmt.Errorf("%w: read stream: %v", domain.ErrAIUnavailable, err)
	}
	result.Content = content.String()
	result.ToolCalls = orderedToolCalls(calls)
	return result, nil
}

// mergeToolCallDeltas assembles streamed tool calls: the first fragment of
// each index carries id and name, later ones append to the arguments.
func mergeToolCallDeltas(calls map[int]*domain.AIToolCall, deltas []wireToolCall) {
	for position, delta := range deltas {
		index := position
		if delta.Index != nil {
			index = *delta.Index
		}
		call, ok := calls[index]
		if !ok {
			call = &domain.AIToolCall{}
			calls[index] = call
		}
		if delta.ID != "" {
			call.ID = delta.ID
		}
		if delta.Function.Name != "" {
			call.Name = delta.Function.Name
		}
		call.Arguments += deref(delta.Function.Arguments)
	}
}

func orderedToolCalls(calls map[int]*domain.AIToolCall) []domain.AIToolCall {
	indexes := make([]int, 0, len(calls))
	for index := range calls {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	ordered := make([]domain.AIToolCall, 0, len(indexes))
	for _, index := range indexes {
		ordered = append(ordered, *calls[index])
	}
	return ordered
}

func (c *Client) Ping(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	c.authorize(request)
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return providerError(response)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	return nil
}

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
