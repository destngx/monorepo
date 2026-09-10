package openai

import (
	"context"
	"encoding/json"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/shared"
)

// CountTokens provides a conservative estimate for OpenAI and OpenAI-
// compatible backends. Unlike Anthropic, these backends do not expose a
// universal preflight count endpoint through this provider abstraction.
func (p *Provider) CountTokens(_ context.Context, req domain.ChatRequest) (domain.TokenCount, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return domain.TokenCount{}, err
	}
	return domain.TokenCount{
		InputTokens: shared.EstimateTokens(string(b)),
		Exact:       false,
		Provider:    p.Name(),
		Model:       req.Model,
	}, nil
}
