package proxy

import (
	"testing"
)

func TestModelMapper_Resolve(t *testing.T) {
	m := NewModelMapper()

	tests := []struct {
		name     string
		model    string
		wantProv string
		wantMod  string
		mapped   bool
	}{
		{"Pattern match Sonnet", "claude-3-5-sonnet-20241022", "github", "gpt-4.1", true},
		{"Pattern match Haiku", "claude-haiku-4-5-20251001", "github", "gpt-4.1", true},
		{"Pattern match Opus", "claude-3-opus-20240229", "github", "gpt-4.1", true},
		{"Generic Claude prefix", "claude-v2", "github", "gpt-4.1", true},
		{"Unknown model passthrough", "my-custom-model", "", "my-custom-model", false},
		{"Empty model default", "", "github", "gpt-4.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target, ok := m.Resolve(tt.model)
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
