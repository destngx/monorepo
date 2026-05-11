package openai

import (
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func (p *Provider) chatCodex(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	resp, err := p.doCodexRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai codex error %d: %s", resp.StatusCode, b)
	}

	content, usage, responseID, created, model, err := parseCodexStream(resp.Body, nil)
	if err != nil {
		return nil, err
	}
	if responseID == "" {
		responseID = "chatcmpl-codex"
	}
	if created == 0 {
		created = time.Now().Unix()
	}
	if model == "" {
		model = req.Model
	}

	return &domain.ChatResponse{
		ID:      responseID,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []domain.Choice{
			{
				Index: 0,
				Message: domain.Message{
					Role:    domain.RoleAssistant,
					Content: content,
				},
				FinishReason: "stop",
			},
		},
		Usage: usage,
	}, nil
}

func (p *Provider) chatCodexStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	resp, err := p.doCodexRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("openai codex stream error %d: %s", resp.StatusCode, b)
	}

	_, usage, _, _, _, err := parseCodexStream(resp.Body, w)
	return usage, err
}

func (p *Provider) doCodexRequest(ctx context.Context, req domain.ChatRequest) (*http.Response, error) {
	body, err := json.Marshal(toCodexResponseRequest(req))
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTURL+pathCodexResponses, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if err := p.setAuthHeaders(httpReq); err != nil {
		return nil, err
	}
	httpReq.Header.Set(headerContentType, contentTypeJSON)
	httpReq.Header.Set(headerOpenAIBeta, codexResponsesExperimental)
	httpReq.Header.Set(headerOriginator, codexOriginator)
	httpReq.Header.Set(headerSessionID, newCodexSessionID())
	httpReq.Header.Set(headerUserAgent, "")
	httpReq.Header.Set(headerVersion, getEnv(envOpenAICodexVersion, codexDefaultVersion))

	return p.client.Do(httpReq)
}

func toCodexResponseRequest(req domain.ChatRequest) codexResponseRequest {
	instructions := make([]string, 0)
	input := make([]codexInputMessage, 0, len(req.Messages))

	for _, msg := range req.Messages {
		if msg.Role == domain.RoleSystem {
			if msg.Content != "" {
				instructions = append(instructions, msg.Content)
			}
			continue
		}

		role := msg.Role
		if role == domain.RoleAssistant || role == domain.RoleTool {
			role = domain.RoleUser
		}
		if role == "" {
			role = domain.RoleUser
		}
		input = append(input, codexInputMessage{
			Role: role,
			Content: []codexInputContent{
				{Type: "input_text", Text: msg.Content},
			},
		})
	}

	if len(instructions) == 0 {
		instructions = append(instructions, "You are a helpful assistant.")
	}
	if len(input) == 0 {
		input = append(input, codexInputMessage{
			Role: domain.RoleUser,
			Content: []codexInputContent{
				{Type: "input_text", Text: ""},
			},
		})
	}

	maxOutputTokens := req.MaxCompletionTokens
	if maxOutputTokens == nil {
		maxOutputTokens = req.MaxTokens
	}

	out := codexResponseRequest{
		Model:           req.Model,
		Instructions:    strings.Join(instructions, "\n\n"),
		Input:           input,
		Stream:          true,
		Store:           false,
		MaxOutputTokens: maxOutputTokens,
	}
	return out
}

func parseCodexStream(body io.Reader, w io.Writer) (string, domain.Usage, string, int64, string, error) {
	var content strings.Builder
	var usage domain.Usage
	var responseID string
	var created int64
	var model string

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		payload := strings.TrimPrefix(line, "data: ")
		var event codexStreamEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		if event.Response != nil {
			if event.Response.ID != "" {
				responseID = event.Response.ID
			}
			if event.Response.CreatedAt != 0 {
				created = event.Response.CreatedAt
			}
			if event.Response.Model != "" {
				model = event.Response.Model
			}
			if event.Response.Usage != nil {
				usage = event.Response.Usage.toDomain()
			}
		}

		switch event.Type {
		case "response.output_text.delta":
			content.WriteString(event.Delta)
			if w != nil {
				if responseID == "" {
					responseID = "chatcmpl-codex"
				}
				if created == 0 {
					created = time.Now().Unix()
				}
				if err := writeOpenAIStreamDelta(w, responseID, created, model, event.Delta); err != nil {
					return content.String(), usage, responseID, created, model, err
				}
			}
		case "response.failed":
			return content.String(), usage, responseID, created, model, fmt.Errorf("openai codex response failed: %s", event.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return content.String(), usage, responseID, created, model, err
	}

	if usage.TotalTokens == 0 {
		usage.CompletionTokens = shared.EstimateTokens(content.String())
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if w != nil {
		shared.InjectUsageChunk(w, usage)
	}

	return content.String(), usage, responseID, created, model, nil
}

func writeOpenAIStreamDelta(w io.Writer, id string, created int64, model string, delta string) error {
	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"delta": map[string]string{
					"content": delta,
				},
				"finish_reason": nil,
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, "data: "+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}

func newCodexSessionID() string {
	return fmt.Sprintf("ai-gateway-%d", time.Now().UnixNano())
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
