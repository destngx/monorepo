package service

import (
	"testing"

	"apps/ai-gateway/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestModelMapper_LeavesGPT54MiniOnGitHubCopilot(t *testing.T) {
	mapper := NewModelMapper()

	target, mapped := mapper.Resolve(domain.ProviderGitHubCopilot, domain.ModelGPT54Mini)

	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderGitHubCopilot, target.Provider)
	assert.Equal(t, domain.ModelGPT54Mini, target.Model)
}

func TestModelMapper_DoesNotRouteUnknownCopilotModelToOpenAI(t *testing.T) {
	mapper := NewModelMapper()

	target, mapped := mapper.Resolve(domain.ProviderGitHubCopilot, "gpt-5.4-nano")

	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderGitHubCopilot, target.Provider)
	assert.Equal(t, "gpt-5.4-nano", target.Model)
}
