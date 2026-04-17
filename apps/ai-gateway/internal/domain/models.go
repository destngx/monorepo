package domain

const (
	ModelDefault          = "gpt-5-mini"
	ModelGPT5Mini         = "gpt-5-mini"
	ModelGPT41            = "gpt-4.1"
	ModelClaudeHaiku      = "claude-haiku-4.5"
	ModelClaudeSonnet     = "claude-sonnet-4.6"
	ModelClaudeOpus       = "claude-opus-4.7"
	ModelClaudeMythos     = "claude-mythos-4.5"
	ModelEmbeddingDefault = "text-embedding-3-small"

	PrefixClaude = "claude-"
)

// ModelInfo represents a single model in OpenAI-compatible list format.
type ModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created,omitempty"`
	OwnedBy string `json:"owned_by,omitempty"`
}

// ModelsResponse is the OpenAI-compatible response for GET /v1/models.
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelInfo `json:"data"`
}
