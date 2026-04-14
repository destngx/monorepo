package domain

// TokenizeRequest is the request for the utility /tokenize endpoint.
type TokenizeRequest struct {
	Text string `json:"text" example:"Hello, world!"`
}

// TokenizeResponse describes the output of the tokenize operation.
type TokenizeResponse struct {
	Tokens []string      `json:"tokens"`
	Stats  TokenizeStats `json:"stats"`
}

// TokenizeStats provides quantitative metrics for the input text.
type TokenizeStats struct {
	TokenCount     int `json:"token_count"`
	CharacterCount int `json:"character_count"`
	ByteCount      int `json:"byte_count"`
}
