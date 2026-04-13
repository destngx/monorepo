package proxy

import "strings"

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
		DefaultTarget: RouteTarget{Provider: "github-copilot", Model: "gpt-4.1"},
	}

	return m
}

// AddExactForProvider adds an exact match that is only valid for a specific provider.
func (m *ModelMapper) AddExactForProvider(provider, model string, target RouteTarget) {
	m.exact[strings.ToLower(provider)+"|"+strings.ToLower(model)] = target
}

// Resolve identifies the target provider and model for an input provider/model pair.
func (m *ModelMapper) Resolve(provider, model string) (RouteTarget, bool) {
	if model == "" {
		return m.DefaultTarget, false
	}

	providerKey := strings.ToLower(provider)
	lowered := strings.ToLower(model)

	// 1. Check exact provider/model match cache (O(1))
	if target, ok := m.exact[providerKey+"|"+lowered]; ok {
		if target.Model == "" {
			target.Model = model
		}
		return target, true
	}

	// 2. Default Claude-family models to GitHub Copilot, preserving the requested model.
	if isClaudeModel(model) {
		return RouteTarget{Provider: "github-copilot", Model: normalizeClaudeModel(model)}, false
	}

	// 3. Passthrough if not mapped.
	return RouteTarget{Provider: m.DefaultTarget.Provider, Model: m.DefaultTarget.Model}, false
}

func isClaudeModel(model string) bool {
	lowered := strings.ToLower(model)
	return strings.HasPrefix(lowered, "claude-")
}

func normalizeClaudeModel(model string) string {
	lowered := strings.ToLower(model)
	switch lowered {
	case "claude-haiku-4-5-20251001":
		return "claude-haiku-4.5"
	default:
		return "claude-haiku-4.5" // fallback to haiku as default claude
	}
}
