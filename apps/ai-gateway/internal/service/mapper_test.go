package service

import (
	"testing"

	"apps/ai-gateway/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestModelMapper_LeavesGPT54MiniOnGitHubCopilot(t *testing.T) {
	mapper := NewModelMapper(domain.ProviderGitHubCopilot)

	target, mapped := mapper.Resolve(domain.ProviderGitHubCopilot, domain.ModelGPT54Mini)

	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderGitHubCopilot, target.Provider)
	assert.Equal(t, domain.ModelGPT54Mini, target.Model)
}

func TestModelMapper_DoesNotRouteUnknownCopilotModelToOpenAI(t *testing.T) {
	mapper := NewModelMapper(domain.ProviderGitHubCopilot)

	target, mapped := mapper.Resolve(domain.ProviderGitHubCopilot, "gpt-5.4-nano")

	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderGitHubCopilot, target.Provider)
	assert.Equal(t, "gpt-5.4-nano", target.Model)
}

func TestModelMapper_RoutesMimoModelsToXiaomiMimo(t *testing.T) {
	mapper := NewModelMapper(domain.ProviderOpenAI)

	// Test with empty provider (should fallback to xiaomi-mimo)
	target, mapped := mapper.Resolve("", "xiaomi-token-plan-sgp/mimo-v2.5")
	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderXiaomiMimo, target.Provider)
	assert.Equal(t, "xiaomi-token-plan-sgp/mimo-v2.5", target.Model)

	// Test with short unscoped name and empty provider
	target2, mapped2 := mapper.Resolve("", "mimo-v2.5")
	assert.False(t, mapped2)
	assert.Equal(t, domain.ProviderXiaomiMimo, target2.Provider)
	assert.Equal(t, "xiaomi-token-plan-sgp/mimo-v2.5", target2.Model)

	// Test with explicit provider (should respect explicit header and not override)
	target3, mapped3 := mapper.Resolve("openai", "mimo-v2.5")
	assert.False(t, mapped3)
	assert.Equal(t, domain.ProviderOpenAI, target3.Provider)
	assert.Equal(t, "xiaomi-token-plan-sgp/mimo-v2.5", target3.Model)
}
