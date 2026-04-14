package domain

// EmbeddingRequest is the OpenAI-compatible request for /v1/embeddings.
type EmbeddingRequest struct {
	Model string `json:"model" example:"text-embedding-3-small"`
	Input any    `json:"input"` // Can be string or []string
}

// EmbeddingResponse is the OpenAI-compatible response for /v1/embeddings.
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  Usage           `json:"usage"`
}

type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}
