package domain

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ChatRequest is the OpenAI-compatible inbound request structure.
// Clients always send this format, regardless of provider.
type ChatRequest struct {
	Model               string         `json:"model"`
	Messages            []Message      `json:"messages"`
	Stream              bool           `json:"stream"`
	StreamOptions       *StreamOptions `json:"stream_options,omitempty"`
	Temperature         *float64       `json:"temperature,omitempty"`
	TopP                *float64       `json:"top_p,omitempty"`
	MaxTokens           *int           `json:"max_tokens,omitempty"`
	Stop                any            `json:"stop,omitempty"`
	N                   *int           `json:"n,omitempty"`
	Tools               []Tool         `json:"tools,omitempty"`
	ToolChoice          any            `json:"tool_choice,omitempty"`
	ReasoningEffort     string         `json:"reasoning_effort,omitempty"`
	MaxCompletionTokens *int           `json:"max_completion_tokens,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type Message struct {
	Role       string     `json:"role"` // "system" | "user" | "assistant" | "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type Tool struct {
	Type     string             `json:"type"` // always "function"
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"` // JSON Schema
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // always "function"
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ChatResponse is the OpenAI-compatible non-streaming response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int       `json:"index"`
	Message      Message   `json:"message,omitempty"`
	Delta        ChatDelta `json:"delta,omitempty"`
	FinishReason string    `json:"finish_reason"`
}

type ChatDelta struct {
	Role      string          `json:"role,omitempty"`
	Content   string          `json:"content,omitempty"`
	ToolCalls []ToolCallChunk `json:"tool_calls,omitempty"`
}

type ToolCallChunk struct {
	Index    int           `json:"index"`
	ID       string        `json:"id,omitempty"`
	Type     string        `json:"type,omitempty"`
	Function FunctionChunk `json:"function,omitempty"`
}

type FunctionChunk struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// Usage carries token counts — always populated in the gateway response.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
