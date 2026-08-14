package anthropic

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"apps/ai-gateway/internal/domain"
)

func TestConvertToAnthropicResponsePreservesClientModel(t *testing.T) {
	resp := &domain.ChatResponse{ID: "response-1", Model: domain.ModelGPT56Luna, Usage: domain.Usage{PromptTokensDetails: &domain.PromptTokensDetails{CachedTokens: 80, CacheWriteTokens: 12}}}

	got := convertToAnthropicResponse(resp, "claude-haiku-4-5-20251001")

	assert.Equal(t, "claude-haiku-4-5-20251001", got.Model)
	assert.Equal(t, 80, got.Usage.CacheReadInputTokens)
	assert.Equal(t, 12, got.Usage.CacheCreationInputTokens)
}
