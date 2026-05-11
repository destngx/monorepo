package anthropic

import (
	"apps/ai-gateway/internal/domain"
	"encoding/json"
)

// ConvertToAnthropicRequest transforms an OpenAI-compatible request into
// Anthropic's Messages API format. Extracts system messages separately.
func (p *Provider) ConvertToAnthropicRequest(req domain.ChatRequest) Request {
	var system string
	messages := make([]Message, 0, len(req.Messages))

	for _, m := range req.Messages {
		if m.Role == roleSystem {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}

		if m.Role == roleTool {
			// OpenAI tool role -> Anthropic tool_result block in a user message
			messages = append(messages, Message{
				Role: roleUser,
				Content: []Content{{
					Type:    typeToolResult,
					ToolID:  m.ToolCallID,
					Content: m.Content,
				}},
			})
			continue
		}

		if m.Role == roleAssistant && len(m.ToolCalls) > 0 {
			// OpenAI assistant message with tool calls -> Anthropic assistant message with tool_use blocks
			content := make([]Content, 0, len(m.ToolCalls)+1)
			if m.Content != "" {
				content = append(content, Content{Type: typeText, Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				content = append(content, Content{
					Type:  typeToolUse,
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			messages = append(messages, Message{
				Role:    roleAssistant,
				Content: content,
			})
			continue
		}

		messages = append(messages, Message{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	maxTokens := 4096
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}

	ar := Request{
		Model:       req.Model,
		MaxTokens:   maxTokens,
		System:      system,
		Messages:    messages,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	// Convert tools
	for _, t := range req.Tools {
		// Handle built-in tools (web_search, code_execution, etc.)
		if t.Type != domain.ToolTypeFunction && t.Type != "" {
			ar.Tools = append(ar.Tools, Tool{
				Type: t.Type,
				Name: t.Type,
			})
			continue
		}

		// Handle standard function tools
		if t.Function != nil {
			ar.Tools = append(ar.Tools, Tool{
				Type:        "function",
				Name:        t.Function.Name,
				Description: t.Function.Description,
				InputSchema: t.Function.Parameters,
			})
		}
	}

	// Convert stop sequences
	if req.Stop != nil {
		switch v := req.Stop.(type) {
		case string:
			ar.StopSeqs = []string{v}
		case []interface{}:
			for _, s := range v {
				if str, ok := s.(string); ok {
					ar.StopSeqs = append(ar.StopSeqs, str)
				}
			}
		}
	}

	return ar
}

func (p *Provider) headers() map[string]string {
	return map[string]string{
		headerAPIKey:           p.apiKey,
		headerAnthropicVersion: apiVersion,
		headerContentType:      contentTypeJSON,
	}
}
