package service

import (
	"apps/ai-gateway/internal/domain"
	"strings"
)

const (
	KeySeparator = "|"
)

// RouteTarget defines the final destination for a request.
type RouteTarget struct {
	Provider string
	Model    string
}

// ModelMapper provides high-performance provider and model pair mapping.
type ModelMapper struct {
	exact         map[string]RouteTarget
	DefaultTarget RouteTarget
}

// NewModelMapper initializes a mapper with standard mappings.
func NewModelMapper() *ModelMapper {
	m := &ModelMapper{
		exact:         make(map[string]RouteTarget),
		DefaultTarget: RouteTarget{Provider: domain.ProviderGitHubCopilot, Model: domain.ModelGPT41},
	}

	return m
}

// AddExactForProvider adds an exact match that is only valid for a specific provider.
func (m *ModelMapper) AddExactForProvider(provider, model string, target RouteTarget) {
	m.exact[strings.ToLower(provider)+KeySeparator+strings.ToLower(model)] = target
}

// Resolve identifies the target provider and model for an input provider/model pair.
func (m *ModelMapper) Resolve(provider, model string) (target RouteTarget, isExact bool) {
	if provider == "" && model == "" {
		return m.DefaultTarget, false
	}

	pKey := strings.ToLower(provider)
	mKey := strings.ToLower(model)

	// 1. O(1) exact mapping check
	if res, ok := m.exact[pKey+KeySeparator+mKey]; ok {
		if res.Model == "" {
			res.Model = model
		}
		return res, true
	}

	// 2. Specific remapping for Claude models on GitHub Copilot (or if unspecified)
	if (pKey == "" || pKey == domain.ProviderGitHubCopilot) && strings.HasPrefix(mKey, domain.PrefixClaude) {
		return RouteTarget{
			Provider: domain.ProviderGitHubCopilot,
			Model:    normalizeClaudeForCopilot(mKey),
		}, false
	}

	// 3. Default to transparent passthrough
	return RouteTarget{Provider: provider, Model: model}, false
}

func normalizeClaudeForCopilot(lowered string) string {
	// TODO: create env to trigger free tier GithubCopilot model, keep current for claude code cli usage
	// if strings.Contains(lowered, "haiku") {
	// 	return domain.ModelClaudeHaiku
	// }
	// if strings.Contains(lowered, "sonnet") {
	// 	return domain.ModelClaudeSonnet
	// }
	// if strings.Contains(lowered, "opus") {
	// 	return domain.ModelClaudeOpus
	// }
	// if strings.Contains(lowered, "mythos") {
	// 	return domain.ModelClaudeMythos
	// }
	return domain.ModelDefault // or use "grok-code-fast-1"
}
