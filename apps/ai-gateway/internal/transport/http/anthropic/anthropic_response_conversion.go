package anthropic

import (
	"encoding/json"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
)

// convertToAnthropicResponse translates an upstream response into the Anthropic
// protocol while retaining the model identifier requested by the client. The
// gateway may route that model to a different upstream model, but exposing the
// routed model makes Claude Code report the wrong model in its usage UI.
func convertToAnthropicResponse(resp *domain.ChatResponse, clientModel string) anthropic.Response {
	model := resp.Model
	if clientModel != "" {
		model = clientModel
	}

	ar := anthropic.Response{
		ID: resp.ID, Type: "message", Role: "assistant", Model: model,
		Content: []anthropic.Content{},
		Usage:   anthropic.Usage{InputTokens: resp.Usage.PromptTokens, OutputTokens: resp.Usage.CompletionTokens},
	}
	if details := resp.Usage.PromptTokensDetails; details != nil {
		ar.Usage.CacheCreationInputTokens = details.CacheWriteTokens
		ar.Usage.CacheReadInputTokens = details.CachedTokens
	}

	if len(resp.Choices) == 0 {
		return ar
	}

	msg := resp.Choices[0].Message
	if msg.Content != "" {
		ar.Content = append(ar.Content, anthropic.Content{Type: "text", Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		var input any
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		name := tc.Function.Name
		if tc.Type != domain.ToolTypeFunction && tc.Type != "" && name == "" {
			name = tc.Type
		}
		ar.Content = append(ar.Content, anthropic.Content{Type: "tool_use", ID: tc.ID, Name: name, Input: normalizeAnthropicToolInput(name, input)})
	}

	ar.StopReason = stopReasonEndTurn
	switch resp.Choices[0].FinishReason {
	case "length":
		ar.StopReason = "max_tokens"
	case "tool_calls":
		ar.StopReason = "tool_use"
	}
	return ar
}
