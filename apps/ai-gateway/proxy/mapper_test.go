package proxy

import (
	"testing"
)

func TestModelMapper_Resolve(t *testing.T) {
	m := NewModelMapper()
	m.AddExactForProvider("github-copilot", "claude-3-5-sonnet-20241022", RouteTarget{"github-copilot", "claude-3-5-sonnet-20241022"})
	m.AddExactForProvider("github-copilot", "claude-haiku-4-5-20251001", RouteTarget{"github-copilot", "claude-haiku-4.5"})
	m.AddExactForProvider("github-copilot", "claude-3-opus-20240229", RouteTarget{"github-copilot", "claude-3-opus-20240229"})
	m.AddExactForProvider("github-copilot", "claude-v2", RouteTarget{"github-copilot", "claude-v2"})

	tests := []struct {
		provider string
		name     string
		model    string
		wantProv string
		wantMod  string
		mapped   bool
	}{
		{"github-copilot", "Exact pair Sonnet", "claude-3-5-sonnet-20241022", "github-copilot", "claude-3-5-sonnet-20241022", true},
		{"github-copilot", "Exact pair Haiku", "claude-haiku-4-5-20251001", "github-copilot", "claude-haiku-4.5", true},
		{"github-copilot", "Exact pair Opus", "claude-3-opus-20240229", "github-copilot", "claude-3-opus-20240229", true},
		{"github-copilot", "Exact pair Claude v2", "claude-v2", "github-copilot", "claude-v2", true},
		{"openai", "Unknown model defaults to gpt-4.1", "my-custom-model", "github-copilot", "gpt-4.1", false},
		{"openai", "Claude model defaults to GitHub Copilot", "claude-v2", "github-copilot", "claude-haiku-4.5", false},
		{"github-copilot", "Empty model default", "", "github-copilot", "gpt-4.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, ok := m.Resolve(tt.provider, tt.model)
			if ok != tt.mapped {
				t.Errorf("Resolve() mapped = %v, want %v", ok, tt.mapped)
			}
			if target.Provider != tt.wantProv {
				t.Errorf("Resolve() provider = %v, want %v", target.Provider, tt.wantProv)
			}
			if target.Model != tt.wantMod {
				t.Errorf("Resolve() model = %v, want %v", target.Model, tt.wantMod)
			}
		})
	}
}
