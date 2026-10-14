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

func TestModelMapper_PassesMimoModelsThroughToDefaultProvider(t *testing.T) {
	mapper := NewModelMapper(domain.ProviderOpenAI)

	target, mapped := mapper.Resolve("", "mimo-v2.5")

	assert.False(t, mapped)
	assert.Equal(t, domain.ProviderOpenAI, target.Provider)
	assert.Equal(t, "mimo-v2.5", target.Model)
}

func TestModelMapper_RoutesClaudeFamiliesToGatewayModels(t *testing.T) {
	mapper := NewModelMapper(domain.ProviderGitHubCopilot)

	tests := []struct {
		name   string
		input  string
		output string
	}{
		{name: "haiku", input: "claude-haiku-4-5-20251001", output: domain.ModelGPT6Luna},
		{name: "sonnet", input: "claude-sonnet-4-6", output: domain.ModelGPT56Terra},
		{name: "opus", input: "claude-opus-4-7", output: domain.ModelGPT61Sol},
		{name: "fable", input: "claude-fable", output: domain.ModelGPT6Astra},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, mapped := mapper.Resolve("", tt.input)
			assert.False(t, mapped)
			assert.Equal(t, domain.ProviderGitHubCopilot, target.Provider)
			assert.Equal(t, tt.output, target.Model)
		})
	}
}
