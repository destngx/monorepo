package openai

import (
	"apps/wealth-management/domain"
	"encoding/base64"
)

type wireFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Arguments   *string        `json:"arguments,omitempty"`
}

type wireToolCall struct {
	Index    *int         `json:"index,omitempty"`
	ID       string       `json:"id,omitempty"`
	Type     string       `json:"type,omitempty"`
	Function wireFunction `json:"function"`
}

type wirePart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    any            `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireTool struct {
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

type wireRequest struct {
	Model          string            `json:"model"`
	Messages       []wireMessage     `json:"messages"`
	Tools          []wireTool        `json:"tools,omitempty"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
	Temperature    *float64          `json:"temperature,omitempty"`
	MaxTokens      int               `json:"max_tokens,omitempty"`
	Stream         bool              `json:"stream,omitempty"`
}

type wireUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

type wireChoice struct {
	Message *struct {
		Content   *string        `json:"content"`
		ToolCalls []wireToolCall `json:"tool_calls"`
	} `json:"message"`
	Delta *struct {
		Content   *string        `json:"content"`
		ToolCalls []wireToolCall `json:"tool_calls"`
	} `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type wireResponse struct {
	Model   string       `json:"model"`
	Choices []wireChoice `json:"choices"`
	Usage   *wireUsage   `json:"usage"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func encodeRequest(request domain.AIRequest, stream bool) wireRequest {
	wire := wireRequest{Model: request.Model, Temperature: request.Temperature, MaxTokens: request.MaxTokens, Stream: stream}
	if request.JSON {
		wire.ResponseFormat = map[string]string{"type": "json_object"}
	}
	for _, tool := range request.Tools {
		wire.Tools = append(wire.Tools, wireTool{Type: "function", Function: wireFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters}})
	}
	for _, message := range request.Messages {
		wire.Messages = append(wire.Messages, encodeMessage(message))
	}
	return wire
}

func encodeMessage(message domain.AIMessage) wireMessage {
	wire := wireMessage{Role: string(message.Role), ToolCallID: message.ToolCallID}
	switch {
	case len(message.Images) > 0:
		parts := []wirePart{}
		if message.Content != "" {
			parts = append(parts, wirePart{Type: "text", Text: message.Content})
		}
		for _, image := range message.Images {
			part := wirePart{Type: "image_url", ImageURL: &struct {
				URL string `json:"url"`
			}{URL: "data:" + image.MediaType + ";base64," + base64.StdEncoding.EncodeToString(image.Data)}}
			parts = append(parts, part)
		}
		wire.Content = parts
	case len(message.ToolCalls) > 0 && message.Content == "":
		wire.Content = nil
	default:
		wire.Content = message.Content
	}
	for _, call := range message.ToolCalls {
		arguments := call.Arguments
		wire.ToolCalls = append(wire.ToolCalls, wireToolCall{ID: call.ID, Type: "function", Function: wireFunction{Name: call.Name, Arguments: &arguments}})
	}
	return wire
}
