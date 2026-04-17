package anthropic

// Request represents the Anthropic Messages API request format.
type Request struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	System      any       `json:"system,omitempty"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
	TopP        *float64  `json:"top_p,omitempty"`
	StopSeqs    []string  `json:"stop_sequences,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	ToolChoice  any       `json:"tool_choice,omitempty"`
}

type Tool struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // Can be string or []Content
}

type Content struct {
	Type    string `json:"type"`                  // "text", "tool_use", "server_tool_use", "web_search_tool_result", etc.
	Text    string `json:"text,omitempty"`        // for text
	ID      string `json:"id,omitempty"`          // for tool_use/server_tool_use
	Name    string `json:"name,omitempty"`        // for tool_use/server_tool_use
	Input   any    `json:"input,omitempty"`       // for tool_use/server_tool_use
	ToolID  string `json:"tool_use_id,omitempty"` // for tool_result/web_search_tool_result
	Content any    `json:"content,omitempty"`     // for tool_result (string or []any)
	IsError bool   `json:"is_error,omitempty"`    // for tool_result
}

// Response represents the Anthropic Messages API response format.
type Response struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // "message"
	Role         string    `json:"role"` // "assistant"
	Model        string    `json:"model"`
	Content      []Content `json:"content"`
	StopReason   string    `json:"stop_reason,omitempty"`
	StopSequence string    `json:"stop_sequence,omitempty"`
	Usage        Usage     `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}
