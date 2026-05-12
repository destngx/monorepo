package domain

const (
	ModelDefault          = ModelMimoV25
	ModelGPT5Mini         = "gpt-5-mini"
	ModelGPT54            = "gpt-5.4"
	ModelGPT54Mini        = "gpt-5.4-mini"
	ModelGPT41            = "gpt-4.1"
	ModelClaudeHaiku      = "claude-haiku-4.5"
	ModelClaudeSonnet     = "claude-sonnet-4.6"
	ModelClaudeOpus       = "claude-opus-4.7"
	ModelClaudeMythos     = "claude-mythos-4.5"
	ModelMimoV25Pro       = "xiaomi-token-plan-sgp/mimo-v2.5-pro"
	ModelMimoV25          = "xiaomi-token-plan-sgp/mimo-v2.5"
	ModelEmbeddingDefault = "text-embedding-3-small"

	PrefixClaude = "claude-"
	PrefixMimo   = "xiaomi-token-plan-sgp/mimo-"
)

const (
	ReasoningEffortNone   = "none"
	ReasoningEffortLow    = "low"
	ReasoningEffortMedium = "medium"
	ReasoningEffortHigh   = "high"
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

// StripMimoScope removes the provider scope prefix from the model name.
func StripMimoScope(model string) string {
	if len(model) > len(PrefixMimo) && model[:len(PrefixMimo)] == PrefixMimo {
		return model[len(PrefixMimo)-len("mimo-"):]
	}
	// Fallback for simple prefix
	if len(model) > len("xiaomi-token-plan-sgp/") && model[:len("xiaomi-token-plan-sgp/")] == "xiaomi-token-plan-sgp/" {
		return model[len("xiaomi-token-plan-sgp/"):]
	}
	return model
}
