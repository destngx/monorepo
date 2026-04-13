package proxy

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"apps/ai-gateway/config"
	"apps/ai-gateway/providers"
)

// Registry maps provider names to their implementations.
type Registry struct {
	providers map[string]providers.Provider
	Mapper    *ModelMapper
	Config    *config.Config
}

// NewRegistry initialises all providers.
// It logs a warning if a provider is missing its configuration.
func NewRegistry(cfg *config.Config) *Registry {
	r := &Registry{
		providers: make(map[string]providers.Provider),
		Mapper:    NewModelMapper(),
		Config:    cfg,
	}

	// Register all providers regardless of config
	r.register(providers.NewRateLimitedProvider(
		providers.NewGitHubCopilot(cfg.GitHubToken),
		cfg.GitHubRate.RPM, cfg.GitHubRate.Burst,
	))

	r.register(providers.NewRateLimitedProvider(
		providers.NewGitHub(cfg.GitHubToken),
		cfg.GitHubRate.RPM, cfg.GitHubRate.Burst,
	))

	r.register(providers.NewRateLimitedProvider(
		providers.NewOpenAI(cfg.OpenAIKey),
		cfg.OpenAIRate.RPM, cfg.OpenAIRate.Burst,
	))

	r.register(providers.NewRateLimitedProvider(
		providers.NewAnthropic(cfg.AnthropicKey),
		cfg.AnthropicRate.RPM, cfg.AnthropicRate.Burst,
	))

	r.register(providers.NewRateLimitedProvider(
		providers.NewOllama(cfg.OllamaBaseURL),
		cfg.OllamaRate.RPM, cfg.OllamaRate.Burst,
	))

	return r
}

func (r *Registry) register(p providers.Provider) {
	r.providers[p.Name()] = p

	// Phase 1: Token Check
	if !p.IsConfigured() {
		log.Printf("[SKIP] Provider %q: missing token", p.Name())
		return
	}

	// Phase 2: Ping Check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		log.Printf("[WARN] Provider %q: token OK but ping FAILED: %v; will return 404 on use", p.Name(), err)
		return
	}

	p.SetReady(true)
	log.Printf("[READY] Provider %q: token OK, ping OK", p.Name())
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

// ResolveRoute determines the actual provider and model for a request.
func (r *Registry) ResolveRoute(httpReq *http.Request, inputModel string) (providers.Provider, string, error) {
	rid, _ := httpReq.Context().Value(requestIDKey).(string)

	if r.Config.Verbose >= 2 {
		log.Printf("[ID:%s] [VERBOSE 2] Resolving route for inputModel: %q", rid, inputModel)
	}

	providerName := httpReq.Header.Get("X-AI-Provider")
	if providerName == "" {
		providerName = r.Mapper.DefaultTarget.Provider
	}
	if providerName == "github" {
		providerName = "github-copilot"
	}

	// 1. Check Smart Mapper using provider + model as the routing key.
	target, mapped := r.Mapper.Resolve(providerName, inputModel)

	if r.Config.Verbose >= 2 {
		log.Printf("[ID:%s] [VERBOSE 2] Smart mapper result: mapped=%v targetProvider=%q targetModel=%q", rid, mapped, target.Provider, target.Model)
	}

	targetModel := target.Model
	if targetModel == "" {
		targetModel = r.Mapper.DefaultTarget.Model
	}

	// 2. If the mapper did not override provider, use the requested provider.
	providerName = target.Provider
	if providerName == "" {
		providerName = httpReq.Header.Get("X-AI-Provider")
		if providerName == "" {
			providerName = r.Mapper.DefaultTarget.Provider
		}
	}
	if providerName == "github" {
		providerName = "github-copilot"
	}

	p, err := r.Get(providerName)
	if err != nil {
		return nil, "", err
	}

	if !p.IsReady() {
		return nil, "", fmt.Errorf("provider %q not ready", providerName)
	}

	return p, targetModel, nil
}
