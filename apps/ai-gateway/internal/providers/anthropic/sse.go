package anthropic

import (
	"apps/ai-gateway/internal/domain"
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// convertStreamToOpenAI reads Anthropic's SSE stream and re-emits it as
// OpenAI-compatible SSE events so the client sees a uniform format.
func (p *Provider) convertStreamToOpenAI(body io.Reader, w io.Writer) (domain.Usage, error) {
	var usage domain.Usage
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024*64), 1024*64)

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, sseDataPrefix) {
			continue
		}
		payload := strings.TrimPrefix(line, sseDataPrefix)

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case eventMessageStart:
			msg, _ := event["message"].(map[string]interface{})
			if u, ok := msg["usage"].(map[string]interface{}); ok {
				if inputTokens, ok := u["input_tokens"].(float64); ok {
					usage.PromptTokens = int(inputTokens)
				}
			}

		case eventContentBlockStart:
			index, _ := event["index"].(float64)
			block, _ := event["content_block"].(map[string]interface{})
			blockType, _ := block["type"].(string)

			if blockType == typeToolUse {
				id, _ := block["id"].(string)
				name, _ := block["name"].(string)
				chunk := map[string]interface{}{
					"object": objectChatCompletionChunk,
					"choices": []map[string]interface{}{{
						"index": 0,
						"delta": map[string]interface{}{
							"tool_calls": []map[string]interface{}{{
								"index":    int(index),
								"id":       id,
								"type":     "function",
								"function": map[string]string{"name": name},
							}},
						},
					}},
				}
				b, _ := json.Marshal(chunk)
				io.WriteString(w, sseDataPrefix+string(b)+"\n\n")
			}

		case eventContentBlockDelta:
			index, _ := event["index"].(float64)
			delta, _ := event["delta"].(map[string]interface{})
			deltaType, _ := delta["type"].(string)

			chunk := map[string]interface{}{
				"object":  objectChatCompletionChunk,
				"choices": []map[string]interface{}{{"index": 0}},
			}

			switch deltaType {
			case typeTextDelta:
				text, _ := delta["text"].(string)
				chunk["choices"].([]map[string]interface{})[0]["delta"] = map[string]string{"content": text}
			case typeInputJSON:
				partial, _ := delta["partial_json"].(string)
				chunk["choices"].([]map[string]interface{})[0]["delta"] = map[string]interface{}{
					"tool_calls": []map[string]interface{}{{
						"index":    int(index),
						"function": map[string]string{"arguments": partial},
					}},
				}
			}

			b, _ := json.Marshal(chunk)
			if _, err := io.WriteString(w, "data: "+string(b)+"\n\n"); err != nil {
				return usage, err
			}
			if f, ok := w.(interface{ Flush() }); ok {
				f.Flush()
			}

		case eventMessageDelta:
			// Extract usage from the final message_delta event
			u, _ := event["usage"].(map[string]interface{})
			if outputTokens, ok := u["output_tokens"].(float64); ok {
				usage.CompletionTokens = int(outputTokens)
			}

		case eventMessageStop:
			usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
			io.WriteString(w, sseDataPrefix+sseDone+"\n\n")
			if f, ok := w.(interface{ Flush() }); ok {
				f.Flush()
			}
		}
	}

	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return usage, scanner.Err()
}
