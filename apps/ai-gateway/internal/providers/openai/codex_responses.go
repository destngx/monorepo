package openai

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"apps/ai-gateway/internal/domain"
)

const maxCodexSSELine = 4 << 20

// responsesCodex serves a non-streaming Responses call over the ChatGPT Codex
// backend, which rejects stream=false: it streams upstream and returns the
// response carried by the terminal event.
func (p *Provider) responsesCodex(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	resp, err := p.doResponsesStreamRequest(ctx, req.WithStream(true))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai responses error %d: %s", resp.StatusCode, b)
	}
	return collectResponsesStream(resp.Body)
}

type responsesStreamEvent struct {
	Type        string         `json:"type"`
	OutputIndex int            `json:"output_index"`
	Item        map[string]any `json:"item"`
	Response    map[string]any `json:"response"`
	Message     string         `json:"message"`
	Error       any            `json:"error"`
}

// collectResponsesStream returns the response of the response.completed (or
// response.incomplete) event. The Codex backend can leave that response's
// output empty, so it is rebuilt from the output_item.done events then.
func collectResponsesStream(body io.Reader) (*domain.ResponsesResponse, error) {
	items := map[int]any{}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), maxCodexSSELine)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var event responsesStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			return nil, fmt.Errorf("decode openai responses stream event: %w", err)
		}
		switch event.Type {
		case "response.output_item.done":
			if event.Item != nil {
				items[event.OutputIndex] = event.Item
			}
		case "response.completed", "response.incomplete":
			if event.Response == nil {
				return nil, fmt.Errorf("openai responses stream: %s event without a response", event.Type)
			}
			if output, _ := event.Response["output"].([]any); len(output) == 0 && len(items) > 0 {
				event.Response["output"] = orderedOutput(items)
			}
			result := domain.ResponsesResponse(event.Response)
			return &result, nil
		case "response.failed", "error":
			return nil, fmt.Errorf("openai responses stream failed: %s", responsesStreamFailure(event))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read openai responses stream: %w", err)
	}
	return nil, errors.New("openai responses stream ended before the response completed")
}

func orderedOutput(items map[int]any) []any {
	indexes := make([]int, 0, len(items))
	for index := range items {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	output := make([]any, 0, len(indexes))
	for _, index := range indexes {
		output = append(output, items[index])
	}
	return output
}

// responsesStreamFailure reads the message of a response.failed or error event.
func responsesStreamFailure(event responsesStreamEvent) string {
	if event.Response != nil {
		if message := errorMessage(event.Response["error"]); message != "" {
			return message
		}
	}
	if event.Message != "" {
		return event.Message
	}
	if message := errorMessage(event.Error); message != "" {
		return message
	}
	return event.Type
}

func errorMessage(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		message, _ := v["message"].(string)
		return message
	}
	return ""
}
