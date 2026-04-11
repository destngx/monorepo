package proxy

import (
	"fmt"

	"apps/ai-gateway/config"
	"apps/ai-gateway/providers"
)

// Registry maps provider names to their implementations.
type Registry struct {
	providers map[string]providers.Provider
}

// NewRegistry initialises all configured providers.
// A provider is registered only if its credentials are present.
func NewRegistry(cfg *config.Config) *Registry {
	r := &Registry{providers: make(map[string]providers.Provider)}

	if cfg.GitHubToken != "" {
		p := providers.NewGitHub(cfg.GitHubToken)
		r.providers[p.Name()] = p
	}
	if cfg.OpenAIKey != "" {
		p := providers.NewOpenAI(cfg.OpenAIKey)
		r.providers[p.Name()] = p
	}
	if cfg.AnthropicKey != "" {
		p := providers.NewAnthropic(cfg.AnthropicKey)
		r.providers[p.Name()] = p
	}
	// Ollama is always available (local, no key needed)
	p := providers.NewOllama(cfg.OllamaBaseURL)
	r.providers[p.Name()] = p

	return r
}

// Get returns the provider for the given name.
func (r *Registry) Get(name string) (providers.Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q — registered: %v", name, r.List())
	}
	return p, nil
}

// List returns the names of all registered providers.
func (r *Registry) List() []string {
	keys := make([]string, 0, len(r.providers))
	for k := range r.providers {
		keys = append(keys, k)
	}
	return keys
}
