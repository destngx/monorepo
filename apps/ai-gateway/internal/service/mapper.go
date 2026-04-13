package service

import "strings"

const (
	ModelGPT41           = "gpt-4.1"
	ModelClaudeHaiku45   = "claude-haiku-4.5"
	ModelClaudeSonnet46  = "claude-sonnet-4.6"
	ModelClaudeOpus46    = "claude-opus-4.6"
	ModelClaudeMyrthos45 = "claude-myrthos-4.5"

	PrefixClaude = "claude-"

	KeySeparator = "|"

	ProviderGitHubCopilot = "github-copilot"
	ProviderGitHub        = "github"
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
		DefaultTarget: RouteTarget{Provider: ProviderGitHubCopilot, Model: ModelGPT41},
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
	if (pKey == "" || pKey == ProviderGitHubCopilot) && strings.HasPrefix(mKey, PrefixClaude) {
		return RouteTarget{
			Provider: ProviderGitHubCopilot,
			Model:    normalizeClaudeForCopilot(mKey),
		}, false
	}

	// 3. Default to transparent passthrough
	return RouteTarget{Provider: provider, Model: model}, false
}

func normalizeClaudeForCopilot(lowered string) string {
	if strings.Contains(lowered, "haiku") {
		return ModelClaudeHaiku45
	}
	if strings.Contains(lowered, "sonnet") {
		return ModelClaudeSonnet46
	}
	if strings.Contains(lowered, "opus") {
		return ModelClaudeOpus46
	}
	if strings.Contains(lowered, "myrthos") {
		return ModelClaudeMyrthos45
	}
	return ModelGPT41 // "grok-code-fast-1"
}
