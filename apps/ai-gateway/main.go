package main

import (
	"encoding/json"
	"log"
	"net/http"

	"apps/ai-gateway/config"
	"apps/ai-gateway/internal/service"
	httptransport "apps/ai-gateway/internal/transport/http"
)

const (
	PathChatCompletions = "/v1/chat/completions"
	PathMessages        = "/v1/messages"
	PathModels          = "/v1/models"
	PathModelsSlash     = "/v1/models/"
	PathEmbeddings      = "/v1/embeddings"
	PathUsage           = "/v1/usage"
	PathHealth          = "/health"

	HeaderContentType = "Content-Type"
	ContentTypeJSON   = "application/json"

	LogFormatListening    = "AI Gateway listening on %s"
	LogFormatRegProviders = "Registered providers: %v"
)

func main() {
	cfg := config.Load()
	registry := service.NewRegistry(cfg)
	openaiHandler := httptransport.NewOpenAIHandler(registry)
	anthroHandler := httptransport.NewAnthropicHandler(registry)
	modelsHandler := httptransport.NewModelsHandler(registry)
	embeddingsHandler := httptransport.NewEmbeddingsHandler(registry)
	usageHandler := httptransport.NewUsageHandler(registry)

	mux := http.NewServeMux()
	mux.Handle(PathChatCompletions, openaiHandler)
	mux.Handle(PathMessages, anthroHandler)
	mux.Handle(PathModels, modelsHandler)
	mux.Handle(PathModelsSlash, modelsHandler)
	mux.Handle(PathEmbeddings, embeddingsHandler)
	mux.Handle(PathUsage, usageHandler)
	mux.Handle(PathHealth, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"providers": registry.List(),
		})
	}))

	stack := httptransport.Chain(mux,
		httptransport.Recovery,
		httptransport.Logger,
		func(next http.Handler) http.Handler {
			return httptransport.CORS(next, "*", "GET, POST, OPTIONS", "Content-Type, X-AI-Provider, Authorization")
		},
	)

	log.Printf(LogFormatListening, cfg.ListenAddr)
	log.Printf(LogFormatRegProviders, registry.List())
	if err := http.ListenAndServe(cfg.ListenAddr, stack); err != nil {
		log.Fatal(err)
	}
}
