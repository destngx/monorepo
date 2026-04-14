package service

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
	"apps/ai-gateway/internal/providers/bedrock"
	"apps/ai-gateway/internal/providers/github_copilot"
	"apps/ai-gateway/internal/providers/github_models"
	"apps/ai-gateway/internal/providers/ollama"
	"apps/ai-gateway/internal/providers/openai"
	"apps/ai-gateway/internal/providers/shared"
)

const (
	logPrefixSkip  = "[SKIP]"
	logPrefixWarn  = "[WARN]"
	logPrefixReady = "[READY]"

	logFormatVerbose2       = "[ID:%s] [VERBOSE 2] %s"
	logMsgResolvingRoute    = "Resolving route for inputModel: %q"
	logMsgSmartMapperResult = "Smart mapper result: mapped=%v targetProvider=%q targetModel=%q"

	errUnknownProvider  = "unknown provider %q — registered: %v"
	errProviderNotReady = "provider %q not ready"
)

// Registry maps provider names to their implementations.
type Registry struct {
	providers map[string]shared.Provider
	Mapper    *ModelMapper
	Config    *config.Config
}

// NewRegistry initialises all providers.
// It logs a warning if a provider is missing its configuration.
func NewRegistry(cfg *config.Config) *Registry {
	r := &Registry{
		providers: make(map[string]shared.Provider),
		Mapper:    NewModelMapper(),
		Config:    cfg,
	}

	// Register all providers regardless of config
	r.register(shared.NewRateLimitedProvider(
		github_copilot.New(cfg.GitHubToken, cfg.Verbose),
		cfg.GitHubRate.RPM, cfg.GitHubRate.Burst,
	))

	r.register(shared.NewRateLimitedProvider(
		github_models.New(cfg.GitHubToken),
		cfg.GitHubRate.RPM, cfg.GitHubRate.Burst,
	))

	r.register(shared.NewRateLimitedProvider(
		openai.New(cfg.OpenAIKey),
		cfg.OpenAIRate.RPM, cfg.OpenAIRate.Burst,
	))

	r.register(shared.NewRateLimitedProvider(
		anthropic.New(cfg.AnthropicKey),
		cfg.AnthropicRate.RPM, cfg.AnthropicRate.Burst,
	))

	r.register(shared.NewRateLimitedProvider(
		ollama.New(cfg.OllamaBaseURL),
		cfg.OllamaRate.RPM, cfg.OllamaRate.Burst,
	))

	if bp, err := bedrock.New(context.Background(), cfg.BedrockRegion); err == nil {
		r.register(shared.NewRateLimitedProvider(
			bp,
			cfg.BedrockRate.RPM, cfg.BedrockRate.Burst,
		))
	} else {
		log.Printf(logPrefixWarn+" Provider %q: failed to initialize: %v", "bedrock", err)
	}

	return r
}

func (r *Registry) register(p shared.Provider) {
	r.providers[p.Name()] = p

	// Phase 1: Token Check
	if !p.IsConfigured() {
		log.Printf(logPrefixSkip+" Provider %q: missing token", p.Name())
		return
	}

	// Phase 2: Ping Check
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := p.Ping(ctx); err != nil {
		log.Printf(logPrefixWarn+" Provider %q: token OK but ping FAILED: %v; will return 404 on use", p.Name(), err)
		return
	}

	p.SetReady(true)
	log.Printf(logPrefixReady+" Provider %q: token OK, ping OK", p.Name())
}

// Get returns the provider for the given name.
func (r *Registry) Get(name string) (shared.Provider, error) {
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf(errUnknownProvider, name, r.List())
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
func (r *Registry) ResolveRoute(httpReq *http.Request, inputModel string) (shared.Provider, string, error) {
	rid, _ := httpReq.Context().Value(domain.RequestIDKey).(string)

	if r.Config.Verbose >= 2 {
		log.Printf(logFormatVerbose2, rid, fmt.Sprintf(logMsgResolvingRoute, inputModel))
	}

	providerName := httpReq.Header.Get(domain.HeaderAIProvider)
	if providerName == "" {
		providerName = r.Mapper.DefaultTarget.Provider
	}

	// 1. Check Smart Mapper using provider + model as the routing key.
	target, mapped := r.Mapper.Resolve(providerName, inputModel)

	if r.Config.Verbose >= 2 {
		log.Printf(logFormatVerbose2, rid, fmt.Sprintf(logMsgSmartMapperResult, mapped, target.Provider, target.Model))
	}

	targetModel := target.Model
	if targetModel == "" {
		targetModel = r.Mapper.DefaultTarget.Model
	}

	// 2. If the mapper did not override provider, use the requested provider.
	providerName = target.Provider
	if providerName == "" {
		providerName = httpReq.Header.Get(domain.HeaderAIProvider)
		if providerName == "" {
			providerName = r.Mapper.DefaultTarget.Provider
		}
	}

	p, err := r.Get(providerName)
	if err != nil {
		return nil, "", err
	}

	if !p.IsReady() {
		return nil, "", fmt.Errorf(errProviderNotReady, providerName)
	}

	return p, targetModel, nil
}
