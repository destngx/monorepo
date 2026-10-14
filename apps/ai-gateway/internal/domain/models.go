package domain

const (
	ModelDefault          = ModelGPT6Luna
	ModelGPT6Astra        = "gpt-6-astra"
	ModelGPT6Luna         = "gpt-6-luna"
	ModelGPT61Sol         = "gpt-6.1-sol"
	ModelGPT56Luna        = "gpt-5.6-luna"
	ModelGPT56Terra       = "gpt-5.6-terra"
	ModelGPT56Sol         = "gpt-5.6-sol"
	ModelGPT55            = "gpt-5.5"
	ModelGPT54            = "gpt-5.4"
	ModelGPT54Mini        = "gpt-5.4-mini"
	ModelGPT5Mini         = "gpt-5-mini"
	ModelGPT41            = "gpt-4.1"
	ModelClaudeHaiku      = "claude-haiku-4.5"
	ModelClaudeSonnet     = "claude-sonnet-5"
	ModelClaudeOpus       = "claude-opus-5.1"
	ModelClaudeMythos     = "claude-mythos-4.8"
	ModelEmbeddingDefault = "text-embedding-3-small"

	PrefixClaude = "claude-"

	ClaudeModelFamilyHaiku  = "haiku"
	ClaudeModelFamilySonnet = "sonnet"
	ClaudeModelFamilyOpus   = "opus"
	ClaudeModelFamilyFable  = "fable"
)

const (
	ReasoningEffortNone   = "none"
	ReasoningEffortLow    = "low"
	ReasoningEffortMedium = "medium"
	ReasoningEffortHigh   = "high"
	ReasoningEffortXHigh  = "xhigh"
	ReasoningEffortMax    = "max"
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
