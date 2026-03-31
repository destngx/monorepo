package main

import (
	copilotAdapter "apps/wealth-management-engine/adapter/ai/copilot"
	skillAIAdapter "apps/wealth-management-engine/adapter/ai/skill"
	toolAIAdapter "apps/wealth-management-engine/adapter/ai/tool"
	cacheAdapter "apps/wealth-management-engine/adapter/cache"
	configAdapter "apps/wealth-management-engine/adapter/config"
	dbAdapter "apps/wealth-management-engine/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management-engine/adapter/fiber"
	fmarketProvider "apps/wealth-management-engine/adapter/market/fmarket"
	vnstockProvider "apps/wealth-management-engine/adapter/market/vnstock"
	mcpAdapter "apps/wealth-management-engine/adapter/mcp"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"apps/wealth-management-engine/service"
	"context"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
	"path/filepath"
)

func main() {
	mcpServer := mcpAdapter.NewServer()
	go func() {
		if err := mcpServer.Run(os.Stdin, os.Stdout); err != nil {
			log.Printf("MCP server stopped: %v", err)
		}
	}()

	configAdapter.Load()

	app := fiber.New()
	ctx := context.Background()

	// Initialize services
	healthSvc := service.NewHealthService()
	sheetsConfig, err := configAdapter.LoadSheetsConfig()
	if err != nil {
		log.Printf("Google Sheets client not configured: %v", err)
	}
	cacheConfig, cacheConfigErr := configAdapter.LoadCacheConfig()
	if cacheConfigErr != nil {
		log.Printf("Upstash Redis client not configured: %v", cacheConfigErr)
	}
	aiConfig, aiConfigErr := configAdapter.LoadAIConfig()
	if aiConfigErr != nil {
		log.Printf("GitHub Copilot client not configured: %v", aiConfigErr)
	}
	fmarketProviderConfig := configAdapter.LoadMarketDataProviderConfig("FMARKET_BASE_URL", "https://api.fmarket.vn")
	vnstockProviderConfig := configAdapter.LoadMarketDataProviderConfig("VNSTOCK_SERVER_URL", "http://localhost:8000")
	var cacheClient port.CacheClient

	// Initialize handlers
	healthHandler := fiberAdapter.NewHealthHandler(healthSvc)
	openAPIHandler := fiberAdapter.NewOpenAPIHandler(app)
	app.Get("/api/health", healthHandler.HealthCheck)
	app.Get("/api/openapi.json", openAPIHandler.Spec)
	app.Get("/api/docs", openAPIHandler.Docs)

	toolIntegrationSvc := service.NewToolIntegrationService(
		toolAIAdapter.NewBalanceTool(),
		skillAIAdapter.NewMarketAnalysisSkill(resolveSkillPath()),
	)
	toolIntegrationHandler := fiberAdapter.NewToolIntegrationHandler(toolIntegrationSvc)
	app.Post("/api/ai/tools", toolIntegrationHandler.Run)

	if err == nil {
		dbClient, clientErr := dbAdapter.NewSheetsClient(ctx, sheetsConfig)
		if clientErr != nil {
			log.Printf("Failed to initialize Google Sheets client: %v", clientErr)
		} else {
			dbSvc := service.NewDatabaseService(dbClient)
			dbHandler := fiberAdapter.NewDatabaseHandler(dbSvc)
			app.Get("/api/sheets/accounts", dbHandler.GetAccounts)
		}
	}

	if cacheConfigErr == nil {
		createdCacheClient, cacheClientErr := cacheAdapter.NewClient(cacheConfig)
		if cacheClientErr != nil {
			log.Printf("Failed to initialize Upstash Redis client: %v", cacheClientErr)
		} else {
			cacheClient = createdCacheClient
			cacheSvc := service.NewCacheService(cacheClient)
			cacheHandler := fiberAdapter.NewCacheHandler(cacheSvc)
			app.Get("/api/cache/health", cacheHandler.Ping)
			app.Post("/api/cache/set", cacheHandler.Set)
			app.Get("/api/cache/get/:key", cacheHandler.Get)
			app.Delete("/api/cache/invalidate", cacheHandler.Invalidate)
		}
	}

	marketProviders := make([]port.MarketProvider, 0, 2)
	fmarket, fmarketErr := fmarketProvider.NewProvider(fmarketProviderConfig)
	if fmarketErr != nil {
		log.Printf("Failed to initialize market provider 'fmarket': %v", fmarketErr)
	} else {
		marketProviders = append(marketProviders, fmarket)
	}
	vnstock, vnstockErr := vnstockProvider.NewProvider(vnstockProviderConfig)
	if vnstockErr != nil {
		log.Printf("Failed to initialize market provider 'vnstock': %v", vnstockErr)
	} else {
		marketProviders = append(marketProviders, vnstock)
	}
	if len(marketProviders) > 0 {
		marketRouting := domain.DefaultMarketRoutingConfig()
		marketProviderSvc := service.NewMarketProviderServiceWithRouting(marketRouting, cacheClient, marketProviders...)
		marketProviderHandler := fiberAdapter.NewMarketProviderHandler(marketProviderSvc)
		marketHandler := fiberAdapter.NewMarketHandler(marketProviderSvc)

		app.Get("/api/external/market/providers/:provider/health", marketProviderHandler.Health)
		app.Get("/api/external/vnstock/health", marketProviderHandler.Health)
		app.Get("/api/external/fmarket/health", marketProviderHandler.Health)

		app.Get("/api/market/ticker", marketHandler.GetTicker)
		app.Get("/api/market/exchange-rate", marketHandler.GetExchangeRate)
		app.Get("/api/market/price-series", marketHandler.GetPriceSeries)
		app.Get("/api/market/bank-rates", marketHandler.GetBankInterestRate)
	}

	if aiConfigErr == nil {
		aiClient, aiClientErr := copilotAdapter.NewClient(aiConfig)
		if aiClientErr != nil {
			log.Printf("Failed to initialize GitHub Copilot client: %v", aiClientErr)
		} else {
			aiSvc := service.NewAIService(aiClient, aiConfig.DefaultModel)
			aiHandler := fiberAdapter.NewAIHandler(aiSvc)
			app.Post("/api/ai/stream", aiHandler.Stream)
			app.Post("/api/ai/json", aiHandler.JSON)
		}
	}

	log.Println("Starting server on :8080")
	log.Fatal(app.Listen(":8080"))
}

func resolveSkillPath() string {
	candidates := []string{
		".agents/skills/market-analysis/SKILL.md",
		"../../.agents/skills/market-analysis/SKILL.md",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Clean("../../.agents/skills/market-analysis/SKILL.md")
}
