package anthropic

import (
	"log/slog"
	"net/http"
	"strings"

	"apps/ai-gateway/internal/domain"
	"apps/ai-gateway/internal/providers/anthropic"
	"apps/ai-gateway/internal/providers/shared"
	"apps/ai-gateway/internal/service"
)

type AnthropicRoute struct {
	Provider        shared.Provider
	Model           string
	ReasoningEffort string
}

type AnthropicRouteInterceptor func(r *http.Request, req anthropic.Request, route AnthropicRoute) (AnthropicRoute, error)

func newAnthropicRouteInterceptor(registry *service.Registry) AnthropicRouteInterceptor {
	switch strings.ToLower(strings.TrimSpace(registry.Config.AnthropicRoute)) {
	case "openai-gpt-5.4-mini-low":
		openAIProvider, err := registry.Get(domain.ProviderOpenAI)
		if err != nil {
			slog.Warn("Anthropic route unavailable; using default route", "route", registry.Config.AnthropicRoute, "error", err)
			return DefaultAnthropicRouteInterceptor
		}
		return OpenAIGPT54MiniLowRouteInterceptor(openAIProvider)
	default:
		return DefaultAnthropicRouteInterceptor
	}
}

func DefaultAnthropicRouteInterceptor(r *http.Request, req anthropic.Request, route AnthropicRoute) (AnthropicRoute, error) {
	return route, nil
}

func OpenAIGPT54MiniLowRouteInterceptor(openAIProvider shared.Provider) AnthropicRouteInterceptor {
	return func(r *http.Request, req anthropic.Request, route AnthropicRoute) (AnthropicRoute, error) {
		route.Provider = openAIProvider
		route.Model = domain.ModelGPT54Mini
		return route, nil
	}
}
