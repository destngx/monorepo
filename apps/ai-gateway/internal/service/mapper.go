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
func NewModelMapper(defaultProvider string) *ModelMapper {
	m := &ModelMapper{
		exact:         make(map[string]RouteTarget),
		DefaultTarget: RouteTarget{Provider: defaultProvider, Model: domain.ModelDefault},
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

	// 2. Specific remapping for Claude models on the default provider (or if unspecified)
	if (pKey == "" || pKey == strings.ToLower(m.DefaultTarget.Provider)) && strings.HasPrefix(mKey, domain.PrefixClaude) {
		return RouteTarget{
			Provider: m.DefaultTarget.Provider,
			Model:    normalizeClaudeForCopilot(mKey),
		}, false
	}

	// 3. Map Mimo models to Xiaomi Mimo provider (Aggressive routing for specific prefix)
	if strings.HasPrefix(mKey, domain.PrefixMimo) || strings.HasPrefix(mKey, "mimo-") {
		targetModel := model
		// If user used short name, normalize to scoped name
		if strings.HasPrefix(mKey, "mimo-") && !strings.HasPrefix(mKey, domain.PrefixMimo) {
			targetModel = "xiaomi-token-plan-sgp/" + model
		}
		return RouteTarget{
			Provider: domain.ProviderXiaomiMimo,
			Model:    targetModel,
		}, false
	}

	// 4. Default to transparent passthrough with fallback to default provider
	targetProvider := provider
	if targetProvider == "" {
		targetProvider = m.DefaultTarget.Provider
	}
	return RouteTarget{Provider: targetProvider, Model: model}, false
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
