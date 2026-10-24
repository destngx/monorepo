// Package openai is an AIClient for the OpenAI Responses API, reached through
// the monorepo ai-gateway or the official endpoint (D10).
package openai

import (
	"apps/wealth-management/domain"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

const (
	providerHeader = "X-AI-Provider"
	maxErrorBody   = 2048
	maxSSELine     = 4 << 20
)

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
	// refusedEfforts are the model and effort pairs the provider refused (a
	// model that does not reason, or not at that effort); later calls leave
	// the effort out.
	refusedEfforts sync.Map
}

// NewClient talks to baseURL (".../v1"). Request deadlines come from the
// caller's context so long streams are not cut by a client-wide timeout.
func NewClient(config domain.AIConfig, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{
		baseURL: strings.TrimRight(config.BaseURL, "/"),
		apiKey:  config.APIKey,
		http:    httpClient,
	}
}

// post sends a request. A refused reasoning effort is sent again without
// it, at the provider's default, and remembered so it is not refused again.
func (c *Client) post(ctx context.Context, body wireRequest) (*http.Response, error) {
	if body.Reasoning == nil {
		return c.send(ctx, body)
	}
	refused := refusalKey(body)
	if _, known := c.refusedEfforts.Load(refused); known {
		body.Reasoning = nil
		return c.send(ctx, body)
	}
	response, err := c.send(ctx, body)
	if errors.Is(err, errReasoningRefused) {
		c.refusedEfforts.Store(refused, struct{}{})
		body.Reasoning = nil
		return c.send(ctx, body)
	}
	return response, err
}

func (c *Client) send(ctx context.Context, body wireRequest) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode ai request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(payload))
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
		err := providerError(response)
		if body.Reasoning != nil && refusesReasoning(response.StatusCode, err) {
			return nil, fmt.Errorf("%w: %w", errReasoningRefused, err)
		}
		return nil, err
	}
	return response, nil
}

func refusalKey(body wireRequest) string {
	return body.Model + "\x00" + body.Reasoning.Effort
}

// errReasoningRefused marks a refused request that names the reasoning
// parameter.
var errReasoningRefused = errors.New("the model does not take a reasoning effort")

// refusesReasoning reads a 400 about the reasoning parameter: from OpenAI as
// is, or from ai-gateway as a 502 that carries the provider's 400.
func refusesReasoning(status int, err error) bool {
	if status != http.StatusBadRequest && status != http.StatusBadGateway {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "reasoning") && (status == http.StatusBadRequest || strings.Contains(message, "400"))
}

func (c *Client) authorize(request *http.Request) {
	if c.apiKey != "" {
		request.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	request.Header.Set(providerHeader, domain.AIProvider)
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
	if message := decoded.errorMessage(); message != "" || decoded.Status == "failed" {
		return domain.AIResponse{}, fmt.Errorf("%w: %s", domain.ErrAIUnavailable, firstNonEmpty(message, "response failed"))
	}
	toolCalls := decoded.toolCalls()
	return domain.AIResponse{
		Model:        firstNonEmpty(decoded.Model, request.Model),
		Content:      decoded.text(),
		ToolCalls:    toolCalls,
		FinishReason: decoded.finishReason(len(toolCalls) > 0),
		Usage:        decoded.usage(),
	}, nil
}

// Stream sends text deltas to onDelta as they arrive. The terminal
// response.completed (or response.incomplete) event carries model, usage and
// status; a stream that ends without one is a provider failure.
func (c *Client) Stream(ctx context.Context, request domain.AIRequest, onDelta func(string) error) (domain.AIResponse, error) {
	wire := encodeRequest(request, true)
	result, err := c.stream(ctx, request, wire, onDelta)
	// ai-gateway reports a refused effort inside the stream, before any text.
	if wire.Reasoning != nil && errors.Is(err, errReasoningRefused) {
		c.refusedEfforts.Store(refusalKey(wire), struct{}{})
		wire.Reasoning = nil
		return c.stream(ctx, request, wire, onDelta)
	}
	return result, err
}

func (c *Client) stream(ctx context.Context, request domain.AIRequest, wire wireRequest, onDelta func(string) error) (domain.AIResponse, error) {
	response, err := c.post(ctx, wire)
	if err != nil {
		return domain.AIResponse{}, err
	}
	defer response.Body.Close()

	var content strings.Builder
	var final *wireResponse
	calls := map[int]*domain.AIToolCall{}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64*1024), maxSSELine)
	for final == nil && scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var event wireEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return domain.AIResponse{}, fmt.Errorf("%w: decode stream event: %v", domain.ErrAIUnavailable, err)
		}
		switch event.Type {
		case "response.output_text.delta":
			if event.Delta == "" {
				continue
			}
			content.WriteString(event.Delta)
			if err := onDelta(event.Delta); err != nil {
				return domain.AIResponse{}, err
			}
		case "response.output_item.added", "response.output_item.done":
			mergeToolCallItem(calls, event.OutputIndex, event.Item)
		case "response.function_call_arguments.delta":
			toolCallAt(calls, event.OutputIndex).Arguments += event.Delta
		case "response.completed", "response.incomplete":
			if event.Response == nil {
				return domain.AIResponse{}, fmt.Errorf("%w: %s event without a response", domain.ErrAIUnavailable, event.Type)
			}
			final = event.Response
		case "response.failed", "error":
			failure := fmt.Errorf("%w: %s", domain.ErrAIUnavailable, streamFailure(event))
			if wire.Reasoning != nil && content.Len() == 0 && len(calls) == 0 && refusesReasoning(http.StatusBadGateway, failure) {
				return domain.AIResponse{}, fmt.Errorf("%w: %w", errReasoningRefused, failure)
			}
			return domain.AIResponse{}, failure
		}
	}
	if err := scanner.Err(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return domain.AIResponse{}, ctxErr
		}
		return domain.AIResponse{}, fmt.Errorf("%w: read stream: %v", domain.ErrAIUnavailable, err)
	}
	if final == nil {
		return domain.AIResponse{}, fmt.Errorf("%w: stream ended before the response completed", domain.ErrAIUnavailable)
	}

	result := domain.AIResponse{Model: firstNonEmpty(final.Model, request.Model), Usage: final.usage()}
	result.Content = content.String()
	if result.Content == "" {
		// Some providers put the whole answer in the terminal event only.
		if result.Content = final.text(); result.Content != "" {
			if err := onDelta(result.Content); err != nil {
				return domain.AIResponse{}, err
			}
		}
	}
	result.ToolCalls = orderedToolCalls(calls)
	if len(result.ToolCalls) == 0 {
		result.ToolCalls = final.toolCalls()
	}
	result.FinishReason = final.finishReason(len(result.ToolCalls) > 0)
	return result, nil
}

// mergeToolCallItem records a function_call item: output_item.added carries
// id and name, output_item.done the complete arguments.
func mergeToolCallItem(calls map[int]*domain.AIToolCall, index int, item *wireItem) {
	if item == nil || item.Type != itemFunctionCall {
		return
	}
	call := toolCallAt(calls, index)
	if item.CallID != "" {
		call.ID = item.CallID
	}
	if item.Name != "" {
		call.Name = item.Name
	}
	if arguments := deref(item.Arguments); arguments != "" {
		call.Arguments = arguments
	}
}

func toolCallAt(calls map[int]*domain.AIToolCall, index int) *domain.AIToolCall {
	call, ok := calls[index]
	if !ok {
		call = &domain.AIToolCall{}
		calls[index] = call
	}
	return call
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

// streamFailure reads the message of a response.failed or error event.
func streamFailure(event wireEvent) string {
	message := event.Message
	if event.Response != nil {
		message = firstNonEmpty(event.Response.errorMessage(), message)
	}
	if event.Error != nil {
		message = firstNonEmpty(message, event.Error.Message)
	}
	return firstNonEmpty(message, event.Type)
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
