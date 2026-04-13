package main

import (
	"encoding/json"
	"log"
	"net/http"

	"apps/ai-gateway/config"
	"apps/ai-gateway/proxy"
)

func main() {
	cfg := config.Load()
	registry := proxy.NewRegistry(cfg)
	openaiHandler := proxy.NewOpenAIHandler(registry)
	anthroHandler := proxy.NewAnthropicHandler(registry)
	modelsHandler := proxy.NewModelsHandler(registry)
	embeddingsHandler := proxy.NewEmbeddingsHandler(registry)
	usageHandler := proxy.NewUsageHandler(registry)

	mux := http.NewServeMux()
	mux.Handle("/v1/chat/completions", openaiHandler)
	mux.Handle("/v1/messages", anthroHandler)
	mux.Handle("/v1/models", modelsHandler)
	mux.Handle("/v1/embeddings", embeddingsHandler)
	mux.Handle("/v1/usage", usageHandler)
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"providers": registry.List(),
		})
	}))

	stack := proxy.Chain(mux,
		proxy.Recovery,
		proxy.Logger,
		proxy.CORS,
	)

	log.Printf("AI Gateway listening on %s", cfg.ListenAddr)
	log.Printf("Registered providers: %v", registry.List())
	if err := http.ListenAndServe(cfg.ListenAddr, stack); err != nil {
		log.Fatal(err)
	}
}
