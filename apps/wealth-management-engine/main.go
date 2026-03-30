package main

import (
	copilotAdapter "apps/wealth-management-engine/adapter/ai/copilot"
	mockAIAdapter "apps/wealth-management-engine/adapter/ai/mock"
	skillAIAdapter "apps/wealth-management-engine/adapter/ai/skill"
	cacheAdapter "apps/wealth-management-engine/adapter/cache"
	configAdapter "apps/wealth-management-engine/adapter/config"
	dbAdapter "apps/wealth-management-engine/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management-engine/adapter/fiber"
	vnstockProvider "apps/wealth-management-engine/adapter/market/vnstock"
	mcpAdapter "apps/wealth-management-engine/adapter/mcp"
	"apps/wealth-management-engine/service"
	"context"
	"github.com/gofiber/fiber/v2"
	"log"
	"os"
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
	vnstockProviderConfig := configAdapter.LoadMarketDataProviderConfig("VNSTOCK_SERVER_URL", "http://localhost:8000")

	// Initialize handlers
	healthHandler := fiberAdapter.NewHealthHandler(healthSvc)
	app.Get("/api/health", healthHandler.HealthCheck)

	toolIntegrationSvc := service.NewToolIntegrationService(
		mockAIAdapter.NewBalanceTool(),
		skillAIAdapter.NewMarketAnalysisSkill(".agents/skills/market-analysis/SKILL.md"),
	)
	toolIntegrationHandler := fiberAdapter.NewToolIntegrationHandler(toolIntegrationSvc)
	app.Post("/api/test/ai/tools", toolIntegrationHandler.Run)

	vnstock, vnstockErr := vnstockProvider.NewProvider(vnstockProviderConfig)
	if vnstockErr != nil {
		log.Printf("Failed to initialize market provider 'vnstock': %v", vnstockErr)
	} else {
		marketProviderSvc := service.NewMarketProviderService(vnstock)
		marketProviderHandler := fiberAdapter.NewMarketProviderHandler(marketProviderSvc)
		app.Get("/api/external/market/providers/:provider/health", marketProviderHandler.Health)
		// Backward-compatible route for existing integrations.
		app.Get("/api/external/vnstock/health", marketProviderHandler.Health)
	}

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
		cacheClient, cacheClientErr := cacheAdapter.NewClient(cacheConfig)
		if cacheClientErr != nil {
			log.Printf("Failed to initialize Upstash Redis client: %v", cacheClientErr)
		} else {
			cacheSvc := service.NewCacheService(cacheClient)
			cacheHandler := fiberAdapter.NewCacheHandler(cacheSvc)
			app.Get("/api/cache/health", cacheHandler.Ping)
			app.Post("/api/cache/set", cacheHandler.Set)
			app.Get("/api/cache/get/:key", cacheHandler.Get)
			app.Delete("/api/cache/invalidate", cacheHandler.Invalidate)
		}
	}

	if aiConfigErr == nil {
		aiClient, aiClientErr := copilotAdapter.NewClient(aiConfig)
		if aiClientErr != nil {
			log.Printf("Failed to initialize GitHub Copilot client: %v", aiClientErr)
		} else {
			aiSvc := service.NewAIService(aiClient, aiConfig.DefaultModel)
			aiHandler := fiberAdapter.NewAIHandler(aiSvc)
			app.Post("/api/test/ai/stream", aiHandler.Stream)
			app.Post("/api/test/ai/json", aiHandler.JSON)
		}
	}

	log.Println("Starting server on :8080")
	log.Fatal(app.Listen(":8080"))
}
