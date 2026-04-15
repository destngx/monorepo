// @title AI Gateway API
// @version 1.0
// @description Multi-provider AI Gateway. Proxies requests to OpenAI, Anthropic, GitHub Copilot, Ollama, Bedrock.
// @contact.name Maintainer
// @contact.email your@email.com
// @license.name MIT
// @BasePath /
// @accept json
// @produce json

package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// Swagger UI and docs
	"apps/ai-gateway/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/logger"
	"apps/ai-gateway/internal/service"
	httptransport "apps/ai-gateway/internal/transport/http"
)

const (
	PathChatCompletions  = "/v1/chat/completions"
	PathMessages         = "/v1/messages"
	PathModelsSlash      = "/v1/models/"
	PathEmbeddings       = "/v1/embeddings"
	PathUsage            = "/v1/usage"
	PathHealth           = "/health"
	PathMetrics          = "/metrics"
	PathMetricsDashboard = "/metrics/dashboard"
	PathMetricsReset     = "/metrics/reset"

	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json"

	LogFormatListening    = "AI Gateway listening on %s"
	LogFormatRegProviders = "Registered providers: %v"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.EnableColor)

	// Make Swagger host dynamic
	docs.SwaggerInfo.Host = ""

	registry := service.NewRegistry(cfg)
	openaiHandler := httptransport.NewOpenAIHandler(registry)
	anthroHandler := httptransport.NewAnthropicHandler(registry)
	modelsHandler := httptransport.NewModelsHandler(registry)
	embeddingsHandler := httptransport.NewEmbeddingsHandler(registry)
	usageHandler := httptransport.NewUsageHandler(registry)
	providersHandler := httptransport.NewProvidersHandler(registry)
	healthHandler := httptransport.NewHealthHandler(registry)
	tokenizeHandler := httptransport.NewTokenizeHandler(registry)

	// Metrics initialization
	collector := service.NewMetricsCollector(cfg.MetricsBufferSize, "tmp/metrics.json", time.Duration(cfg.MetricsSaveInterval)*time.Second)
	collector.LoadFromDisk()
	go collector.StartPersistence(context.Background())

	metricsHandler := httptransport.NewMetricsHandler(collector)
	dashboardHandler := httptransport.NewDashboardHandler()
	resetHandler := httptransport.NewMetricsResetHandler(collector)

	// Graceful shutdown flush
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Shutting down... flushing metrics")
		collector.Flush()
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	mux.Handle(PathChatCompletions, openaiHandler)
	mux.Handle(PathMessages, anthroHandler)
	mux.Handle(PathModelsSlash, modelsHandler)
	mux.Handle(PathEmbeddings, embeddingsHandler)
	mux.Handle(PathUsage, usageHandler)
	mux.Handle("/tokenize", tokenizeHandler)

	mux.Handle("/providers", providersHandler)

	// Swagger documentation
	mux.Handle("/docs/", httpSwagger.WrapHandler)

	mux.Handle(PathHealth, healthHandler)
	mux.Handle(PathMetrics, metricsHandler)
	mux.Handle(PathMetricsDashboard, dashboardHandler)
	mux.Handle(PathMetricsReset, resetHandler)

	stack := httptransport.Chain(mux,
		httptransport.Recovery,
		httptransport.Logger,
		httptransport.Metrics(collector),
		func(next http.Handler) http.Handler {
			return httptransport.CORS(next, "*", "GET, POST, OPTIONS", "Content-Type, X-AI-Provider, Authorization")
		},
	)

	slog.Info(LogFormatListening, "addr", cfg.ListenAddr)
	slog.Info(LogFormatRegProviders, "providers", registry.List())
	if err := http.ListenAndServe(cfg.ListenAddr, stack); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
