package main

import (
	openaiAdapter "apps/wealth-management/adapter/ai/openai"
	cacheAdapter "apps/wealth-management/adapter/cache"
	configAdapter "apps/wealth-management/adapter/config"
	dbAdapter "apps/wealth-management/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management/adapter/fiber"
	loggerAdapter "apps/wealth-management/adapter/logger"
	coingeckoProvider "apps/wealth-management/adapter/market/coingecko"
	currencyapiProvider "apps/wealth-management/adapter/market/currencyapi"
	fmarketProvider "apps/wealth-management/adapter/market/fmarket"
	vnstockProvider "apps/wealth-management/adapter/market/vnstock"
	yahooProvider "apps/wealth-management/adapter/market/yahoo"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	duckduckgoSearch "apps/wealth-management/adapter/search/duckduckgo"
	exaSearch "apps/wealth-management/adapter/search/exa"
	tavilySearch "apps/wealth-management/adapter/search/tavily"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	aiService "apps/wealth-management/service/ai"
	cacheService "apps/wealth-management/service/cache"
	marketProviderService "apps/wealth-management/service/market_provider"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// Load .env.local first so LOG_LEVEL and the rest of the config come from it.
	configAdapter.Load()
	logLevel, colorEnabled := configAdapter.LoadLoggerConfig()
	log := loggerAdapter.NewLogger(logLevel, colorEnabled)
	ctx := context.Background()

	serverConfig, err := configAdapter.LoadServerConfig()
	if err != nil {
		log.LogError(ctx, "invalid server configuration", err)
		os.Exit(1)
	}
	cardConfig, err := configAdapter.LoadCardConfig()
	if err != nil {
		log.LogError(ctx, "invalid card programs configuration", err)
		os.Exit(1)
	}
	captureLimits, err := configAdapter.LoadCaptureLimits()
	if err != nil {
		log.LogError(ctx, "invalid Smart Capture configuration", err)
		os.Exit(1)
	}

	health := service.NewHealthService()
	mcpServer := mcpAdapter.NewServer(health, log)
	mcpServer.SetMaxRequestBytes(bodyLimit(captureLimits))

	// No startup banner: stdout carries the MCP stdio stream. Immutable keeps
	// request strings valid after a handler returns (they reach caches and logs).
	app := fiber.New(fiber.Config{DisableStartupMessage: true, Immutable: true, BodyLimit: bodyLimit(captureLimits)})
	app.Use(fiberAdapter.RequestIDMiddleware(log))
	app.Use(fiberAdapter.LoggingMiddleware(log))
	app.Use(fiberAdapter.RecoveryMiddleware(log))

	guard := fiberAdapter.NewWriteGuard(serverConfig.APIToken, serverConfig.Host)
	if serverConfig.APIToken == "" {
		log.LogApplicationEvent(ctx, "API_TOKEN is not set: write endpoints only work when HOST is a loopback address")
	}
	openAPIHandler := fiberAdapter.NewOpenAPIHandler(app)
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")
	api.Get("/health", fiberAdapter.NewHealthHandler(health).HealthCheck, fiberAdapter.RouteMeta{Summary: "Reports the status of each dependency (sheets, cache, market providers). 503 when any is down or unconfigured.", Tags: []string{"health"}})
	app.Get("/api/openapi.json", openAPIHandler.Spec)
	app.Get("/api/docs", openAPIHandler.Docs)

	cacheClient, cacheSvc := setupCache(ctx, log, health, api)
	dbSvc, analyticsSvc := setupSheets(ctx, log, health, api, guard, mcpServer, cacheSvc, cardConfig)
	marketSvc := setupMarket(ctx, log, health, api, mcpServer, cacheClient)
	aiOptions := aiService.Options{CaptureLimits: captureLimits, Search: setupSearch(ctx, log, cacheSvc)}
	if marketSvc != nil {
		analysisSvc := service.NewMarketAnalysisService(marketSvc, nil)
		fiberAdapter.RegisterMarketAnalysisRoutes(api, fiberAdapter.NewMarketAnalysisHandler(analysisSvc, log))
		aiOptions.Market = analysisSvc
	}
	if dbSvc != nil && marketSvc != nil {
		portfolioSvc := service.NewPortfolioService(dbSvc, marketSvc, nil)
		fiberAdapter.RegisterPortfolioRoutes(api, fiberAdapter.NewPortfolioHandler(portfolioSvc, log))
		aiOptions.Portfolio = portfolioSvc
	}
	setupAI(ctx, log, health, api, guard, mcpServer, dbSvc, analyticsSvc, cacheSvc, aiOptions)

	// Mounted last so every API route above keeps priority over the UI.
	if !fiberAdapter.MountUI(app, fiberAdapter.DefaultUIBuildDir) {
		log.LogApplicationEvent(ctx, "UI build not found at "+fiberAdapter.DefaultUIBuildDir+"; run `pnpx nx run wealth-management:build-ui` to serve the UI")
	}

	go func() {
		if err := mcpServer.Run(os.Stdin, os.Stdout); err != nil {
			log.LogError(ctx, "mcp server stopped", err)
		}
	}()

	address := net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port))
	log.LogApplicationEvent(ctx, "starting server on "+address)
	if err := app.Listen(address); err != nil {
		log.LogError(ctx, "server failed to start", err)
		os.Exit(1)
	}
}

func setupCache(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar) (port.CacheClient, port.CacheService) {
	cacheConfig, err := configAdapter.LoadCacheConfig()
	if err != nil {
		log.LogError(ctx, "cache disabled: reads go straight to the sheet", err)
		health.Register(port.HealthCheck{Name: "cache"})
		return nil, nil
	}
	client, err := cacheAdapter.NewClient(cacheConfig, log)
	if err != nil {
		log.LogError(ctx, "failed to initialize upstash redis client", err)
		health.Register(port.HealthCheck{Name: "cache"})
		return nil, nil
	}
	cacheSvc := cacheService.New(client, log)
	health.Register(port.HealthCheck{Name: "cache", Probe: cacheSvc.Ping})
	api.Group("/cache", "cache").Get("/health", fiberAdapter.NewCacheHandler(cacheSvc, log).Ping, fiberAdapter.RouteMeta{Summary: "Checks the cache backend."})
	return client, cacheSvc
}

// setupSheets returns nil services when the sheet is not configured.
func setupSheets(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, guard fiberAdapter.WriteGuard, mcpServer *mcpAdapter.Server, cacheSvc port.CacheService, cards domain.CardConfig) (port.DatabaseService, port.AnalyticsService) {
	sheetsConfig, err := configAdapter.LoadSheetsConfig()
	if err != nil {
		log.LogError(ctx, "sheet features disabled", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		return nil, nil
	}
	client, err := dbAdapter.NewSheetsClient(ctx, sheetsConfig, log)
	if err != nil {
		log.LogError(ctx, "failed to initialize google sheets client", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		return nil, nil
	}
	if configAdapter.LoadTestMode() {
		// WM-TEST-2: a test-mode engine must never touch the real sheet.
		if err := checkTestSheet(sheetsConfig.SpreadsheetID, client.SpreadsheetTitle); err != nil {
			log.LogError(ctx, "WM_TEST_MODE refuses this spreadsheet", err)
			os.Exit(1)
		}
		log.LogApplicationEvent(ctx, "test mode: using a disposable spreadsheet copy")
	}
	dbSvc := service.NewDatabaseService(client, cacheSvc, log, service.DatabaseServiceOptions{BudgetSheetTemplate: configAdapter.BudgetSheetTemplate()})
	analyticsSvc := service.NewAnalyticsService(dbSvc, cards, nil)
	health.Register(port.HealthCheck{Name: "sheets", Probe: func(context.Context) error { return dbSvc.Ping() }})
	mcpServer.SetDatabaseService(dbSvc)
	mcpServer.SetAnalyticsService(analyticsSvc)
	fiberAdapter.RegisterSheetRoutes(api, fiberAdapter.NewDatabaseHandler(dbSvc, log), guard)
	fiberAdapter.RegisterAnalyticsRoutes(api, fiberAdapter.NewAnalyticsHandler(analyticsSvc, log))
	return dbSvc, analyticsSvc
}

// checkTestSheet allows only a disposable copy; the harness separately
// checks the copy id against the production id.
func checkTestSheet(spreadsheetID string, title func() (string, error)) error {
	name, err := title()
	if err != nil {
		return fmt.Errorf("read spreadsheet title: %w", err)
	}
	return domain.CheckTestSheet(spreadsheetID, name, "")
}

// setupAI wires the AI use cases (§10). They read prompts and data from the
// sheet, so they need it; the status route stays up to report what is missing.
func setupAI(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, guard fiberAdapter.WriteGuard, mcpServer *mcpAdapter.Server, dbSvc port.DatabaseService, analyticsSvc port.AnalyticsService, cacheSvc port.CacheService, options aiService.Options) {
	if dbSvc == nil {
		log.LogApplicationEvent(ctx, "AI features disabled: they need the Google Sheet")
		return
	}
	aiConfig, err := configAdapter.LoadAIConfig()
	if err != nil {
		log.LogError(ctx, "AI features disabled", err)
		health.Register(port.HealthCheck{Name: "ai"})
	}
	client := openaiAdapter.NewClient(aiConfig, nil)
	if err == nil {
		health.Register(port.HealthCheck{Name: "ai", Probe: client.Ping})
	}
	aiSvc := aiService.New(client, dbSvc, analyticsSvc, cacheSvc, aiConfig, log, options)
	mcpServer.SetAIService(aiSvc)
	fiberAdapter.RegisterAIRoutes(api, fiberAdapter.NewAIHandler(aiSvc, log).WithCaptureLimits(options.CaptureLimits), guard)
}

// setupSearch chains the web search providers (WM-AI-12): Tavily and Exa
// when their keys are set, then DuckDuckGo, which needs none.
func setupSearch(ctx context.Context, log *loggerAdapter.Logger, cacheSvc port.CacheService) port.SearchService {
	config := configAdapter.LoadSearchConfig()
	providers := make([]port.SearchProvider, 0, 3)
	if config.TavilyAPIKey != "" {
		if provider, err := tavilySearch.NewProvider(tavilySearch.DefaultBaseURL, config.TavilyAPIKey); err != nil {
			log.LogError(ctx, "search provider disabled", err)
		} else {
			providers = append(providers, provider)
		}
	}
	if config.ExaAPIKey != "" {
		if provider, err := exaSearch.NewProvider(exaSearch.DefaultBaseURL, config.ExaAPIKey); err != nil {
			log.LogError(ctx, "search provider disabled", err)
		} else {
			providers = append(providers, provider)
		}
	}
	providers = append(providers, duckduckgoSearch.NewProvider(duckduckgoSearch.DefaultBaseURL))
	searchSvc := service.NewSearchService(cacheSvc, log, providers...)
	log.LogApplicationEvent(ctx, "web search providers: "+strings.Join(searchSvc.Providers(), ", "))
	return searchSvc
}

// bodyLimit fits a full Smart Capture batch: base64 inflates images by 4/3
// and the JSON envelope needs headroom. Other endpoints stay well below it.
func bodyLimit(limits domain.CaptureLimits) int {
	const minimum = 4 << 20
	return max(minimum, limits.MaxBatchBytes/3*4+(1<<20))
}

// setupMarket registers every market data provider (WM-MKT-1). A provider
// whose config is invalid is skipped and the routing chains fall through to
// the next one. It returns nil when no provider could be built.
func setupMarket(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, mcpServer *mcpAdapter.Server, cacheClient port.CacheClient) port.MarketProviderService {
	providers := make([]port.MarketProvider, 0, 5)
	add := func(provider port.MarketProvider, err error) {
		if err != nil {
			log.LogError(ctx, "market provider disabled", err)
			return
		}
		providers = append(providers, provider)
	}

	fmarket, err := fmarketProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("FMARKET_BASE_URL", "https://api.fmarket.vn"), log)
	add(fmarket, err)
	add(vnstockProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("VNSTOCK_SERVER_URL", "http://localhost:8000"), log))
	coingeckoConfig, coingeckoKey := configAdapter.LoadCoinGeckoConfig(coingeckoProvider.DefaultBaseURL)
	add(coingeckoProvider.NewProvider(coingeckoConfig, coingeckoKey, log))
	add(currencyapiProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("CURRENCY_API_BASE_URL", currencyapiProvider.DefaultBaseURL), log))
	add(yahooProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("YAHOO_FINANCE_BASE_URL", yahooProvider.DefaultBaseURL), log))

	for _, provider := range providers {
		health.Register(port.HealthCheck{Name: "market:" + provider.Name(), Probe: func(ctx context.Context) error {
			_, err := provider.Health(ctx)
			return err
		}})
	}
	if len(providers) == 0 {
		return nil
	}
	marketSvc := marketProviderService.NewServiceWithRouting(domain.DefaultMarketRoutingConfig(), cacheClient, log, providers...)
	mcpServer.SetMarketService(marketSvc)
	fiberAdapter.RegisterMarketRoutes(api, fiberAdapter.NewMarketHandler(marketSvc, log), fiberAdapter.NewMarketProviderHandler(marketSvc, log))
	if fmarket != nil {
		fmarketSvc := fmarketProvider.NewService(fmarket, cacheClient)
		mcpServer.SetFmarketService(fmarketSvc)
		fiberAdapter.RegisterFmarketRoutes(api, fiberAdapter.NewFmarketHandler(fmarketSvc, marketSvc, log))
	}
	return marketSvc
}
