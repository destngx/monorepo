package github_copilot

import (
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func (p *Provider) chatResponsesStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	start := time.Now()
	p.vlogf(1, "[github-copilot] responses stream start model=%q", req.Model)

	httpReq, err := p.newResponsesRequest(ctx, req)
	if err != nil {
		return domain.Usage{}, err
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return domain.Usage{}, err
	}
	defer resp.Body.Close()
	p.vlogf(1, "[github-copilot] upstream responses call took=%s status=%d", time.Since(start), resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return domain.Usage{}, fmt.Errorf("github copilot responses error %d: %s", resp.StatusCode, b)
	}

	return transformResponsesStream(resp.Body, w, req.Model)
}

func transformResponsesStream(body io.Reader, w io.Writer, fallbackModel string) (domain.Usage, error) {
	var usage domain.Usage
	var responseID string
	var created int64
	model := fallbackModel
	var completionTokens int

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)
		if payload == sseDone {
			break
		}

		var event responsesStreamEvent
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
		case "response.output_text.delta", "response.text.delta", "response.content_part.delta":
			completionTokens += shared.EstimateTokens(event.Delta)
			if responseID == "" {
				responseID = "chatcmpl-copilot-responses"
			}
			if created == 0 {
				created = time.Now().Unix()
			}
			if err := writeChatCompletionDelta(w, responseID, created, model, event.Delta, ""); err != nil {
				return usage, err
			}
		case "response.reasoning_text.delta":
			completionTokens += shared.EstimateTokens(event.Delta)
			if responseID == "" {
				responseID = "chatcmpl-copilot-responses"
			}
			if created == 0 {
				created = time.Now().Unix()
			}
			if err := writeChatCompletionDelta(w, responseID, created, model, "", event.Delta); err != nil {
				return usage, err
			}
		case "response.failed":
			return usage, fmt.Errorf("github copilot responses failed: %s", event.Error)
		}
	}
	if err := scanner.Err(); err != nil {
		return usage, err
	}

	if usage.TotalTokens == 0 {
		usage.CompletionTokens = completionTokens
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	shared.InjectUsageChunk(w, usage)
	return usage, nil
}

func writeChatCompletionDelta(w io.Writer, id string, created int64, model string, content string, reasoningContent string) error {
	delta := map[string]string{}
	if content != "" {
		delta["content"] = content
	}
	if reasoningContent != "" {
		delta["reasoning_content"] = reasoningContent
	}

	chunk := map[string]interface{}{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]interface{}{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": nil,
			},
		},
	}
	b, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	if _, err := io.WriteString(w, sseDataPrefix+string(b)+"\n\n"); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() }); ok {
		f.Flush()
	}
	return nil
}
