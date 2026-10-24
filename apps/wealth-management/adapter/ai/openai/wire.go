package openai

import (
	"apps/wealth-management/domain"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Responses API item and content types.
const (
	itemMessage            = "message"
	itemFunctionCall       = "function_call"
	itemFunctionCallOutput = "function_call_output"
	contentInputText       = "input_text"
	contentInputImage      = "input_image"
	contentOutputText      = "output_text"
)

type wireContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// wireItem is one input or output item: a message, a function call the model
// made, or the output of that call.
type wireItem struct {
	Type      string        `json:"type"`
	Role      string        `json:"role,omitempty"`
	Content   []wireContent `json:"content,omitempty"`
	CallID    string        `json:"call_id,omitempty"`
	Name      string        `json:"name,omitempty"`
	Arguments *string       `json:"arguments,omitempty"`
	Output    *string       `json:"output,omitempty"`
}

// wireTool is a function tool. Strict is sent as false because the Responses
// API defaults it to true, which rejects schemas with optional properties.
type wireTool struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      bool           `json:"strict"`
}

type wireReasoning struct {
	Effort string `json:"effort"`
}

type wireText struct {
	Format struct {
		Type string `json:"type"`
	} `json:"format"`
}

// wireRequest is a POST /responses body. Store is always false: every call
// sends the full history, so the provider has no reason to keep it.
type wireRequest struct {
	Model           string         `json:"model"`
	Instructions    string         `json:"instructions,omitempty"`
	Input           []wireItem     `json:"input"`
	Tools           []wireTool     `json:"tools,omitempty"`
	Text            *wireText      `json:"text,omitempty"`
	Reasoning       *wireReasoning `json:"reasoning,omitempty"`
	Temperature     *float64       `json:"temperature,omitempty"`
	MaxOutputTokens int            `json:"max_output_tokens,omitempty"`
	Stream          bool           `json:"stream,omitempty"`
	Store           bool           `json:"store"`
}

type wireUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// wireError is an error object, or the bare string ai-gateway sends in its
// own response.failed stream event.
type wireError struct {
	Message string `json:"message"`
}

func (e *wireError) UnmarshalJSON(raw []byte) error {
	var message string
	if json.Unmarshal(raw, &message) == nil {
		e.Message = message
		return nil
	}
	var object struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	e.Message = object.Message
	return nil
}

type wireResponse struct {
	Model             string `json:"model"`
	Status            string `json:"status"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []wireItem `json:"output"`
	Usage  *wireUsage `json:"usage"`
	Error  *wireError `json:"error"`
}

// wireEvent is one streamed event; which fields are set depends on Type.
type wireEvent struct {
	Type        string        `json:"type"`
	OutputIndex int           `json:"output_index"`
	Delta       string        `json:"delta"`
	Item        *wireItem     `json:"item"`
	Response    *wireResponse `json:"response"`
	Message     string        `json:"message"`
	Error       *wireError    `json:"error"`
}

func encodeRequest(request domain.AIRequest, stream bool) wireRequest {
	wire := wireRequest{Model: request.Model, Input: []wireItem{}, Temperature: request.Temperature, MaxOutputTokens: request.MaxTokens, Stream: stream}
	if request.Effort != "" {
		wire.Reasoning = &wireReasoning{Effort: string(request.Effort)}
	}
	if request.JSON {
		wire.Text = &wireText{}
		wire.Text.Format.Type = "json_object"
	}
	for _, tool := range request.Tools {
		wire.Tools = append(wire.Tools, wireTool{Type: "function", Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters})
	}
	// System messages become instructions, as the Responses API intends; the
	// ChatGPT Codex backend behind ai-gateway rejects system-role input items.
	var instructions []string
	for _, message := range request.Messages {
		if message.Role == domain.AIRoleSystem {
			instructions = append(instructions, message.Content)
			continue
		}
		wire.Input = append(wire.Input, encodeMessage(message)...)
	}
	wire.Instructions = strings.Join(instructions, "\n\n")
	return wire
}

// encodeMessage maps a chat message to input items: a tool result becomes a
// function_call_output, and an assistant's tool calls become function_call
// items after its text.
func encodeMessage(message domain.AIMessage) []wireItem {
	if message.Role == domain.AIRoleTool {
		output := message.Content
		return []wireItem{{Type: itemFunctionCallOutput, CallID: message.ToolCallID, Output: &output}}
	}
	items := []wireItem{}
	if message.Role != domain.AIRoleAssistant || message.Content != "" {
		items = append(items, wireItem{Type: itemMessage, Role: string(message.Role), Content: encodeContent(message)})
	}
	for _, call := range message.ToolCalls {
		arguments := call.Arguments
		items = append(items, wireItem{Type: itemFunctionCall, CallID: call.ID, Name: call.Name, Arguments: &arguments})
	}
	return items
}

func encodeContent(message domain.AIMessage) []wireContent {
	if message.Role == domain.AIRoleAssistant {
		return []wireContent{{Type: contentOutputText, Text: message.Content}}
	}
	parts := []wireContent{}
	if message.Content != "" || len(message.Images) == 0 {
		parts = append(parts, wireContent{Type: contentInputText, Text: message.Content})
	}
	for _, image := range message.Images {
		parts = append(parts, wireContent{Type: contentInputImage, ImageURL: "data:" + image.MediaType + ";base64," + base64.StdEncoding.EncodeToString(image.Data)})
	}
	return parts
}

// text joins the output_text of every message item.
func (r wireResponse) text() string {
	var text strings.Builder
	for _, item := range r.Output {
		if item.Type != itemMessage {
			continue
		}
		for _, content := range item.Content {
			if content.Type == contentOutputText {
				text.WriteString(content.Text)
			}
		}
	}
	return text.String()
}

func (r wireResponse) toolCalls() []domain.AIToolCall {
	var calls []domain.AIToolCall
	for _, item := range r.Output {
		if item.Type == itemFunctionCall {
			calls = append(calls, domain.AIToolCall{ID: item.CallID, Name: item.Name, Arguments: deref(item.Arguments)})
		}
	}
	return calls
}

func (r wireResponse) usage() domain.AIUsage {
	if r.Usage == nil {
		return domain.AIUsage{}
	}
	return domain.AIUsage{PromptTokens: r.Usage.InputTokens, CompletionTokens: r.Usage.OutputTokens}
}

// finishReason keeps the chat completions vocabulary callers already know.
func (r wireResponse) finishReason(hasToolCalls bool) string {
	switch {
	case hasToolCalls:
		return "tool_calls"
	case r.Status == "incomplete" && r.IncompleteDetails != nil:
		if r.IncompleteDetails.Reason == "max_output_tokens" {
			return "length"
		}
		return r.IncompleteDetails.Reason
	default:
		return "stop"
	}
}

func (r wireResponse) errorMessage() string {
	if r.Error != nil && r.Error.Message != "" {
		return r.Error.Message
	}
	return ""
}
