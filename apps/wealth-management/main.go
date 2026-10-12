package main

import (
	cacheAdapter "apps/wealth-management/adapter/cache"
	configAdapter "apps/wealth-management/adapter/config"
	dbAdapter "apps/wealth-management/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management/adapter/fiber"
	loggerAdapter "apps/wealth-management/adapter/logger"
	fmarketProvider "apps/wealth-management/adapter/market/fmarket"
	vnstockProvider "apps/wealth-management/adapter/market/vnstock"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	cacheService "apps/wealth-management/service/cache"
	marketProviderService "apps/wealth-management/service/market_provider"
	"context"
	"net"
	"os"
	"strconv"

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

	health := service.NewHealthService()
	mcpServer := mcpAdapter.NewServer(health, log)

	// No startup banner: stdout carries the MCP stdio stream. Immutable keeps
	// request strings valid after a handler returns (they reach caches and logs).
	app := fiber.New(fiber.Config{DisableStartupMessage: true, Immutable: true})
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
	setupSheets(ctx, log, health, api, guard, mcpServer, cacheSvc, cardConfig)
	setupMarket(ctx, log, health, api, mcpServer, cacheClient)

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

func setupSheets(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, guard fiberAdapter.WriteGuard, mcpServer *mcpAdapter.Server, cacheSvc port.CacheService, cards domain.CardConfig) {
	sheetsConfig, err := configAdapter.LoadSheetsConfig()
	if err != nil {
		log.LogError(ctx, "sheet features disabled", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		return
	}
	client, err := dbAdapter.NewSheetsClient(ctx, sheetsConfig, log)
	if err != nil {
		log.LogError(ctx, "failed to initialize google sheets client", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		return
	}
	dbSvc := service.NewDatabaseService(client, cacheSvc, log, service.DatabaseServiceOptions{BudgetSheetTemplate: configAdapter.BudgetSheetTemplate()})
	analyticsSvc := service.NewAnalyticsService(dbSvc, cards, nil)
	health.Register(port.HealthCheck{Name: "sheets", Probe: func(context.Context) error { return dbSvc.Ping() }})
	mcpServer.SetDatabaseService(dbSvc)
	mcpServer.SetAnalyticsService(analyticsSvc)
	fiberAdapter.RegisterSheetRoutes(api, fiberAdapter.NewDatabaseHandler(dbSvc, log), guard)
	fiberAdapter.RegisterAnalyticsRoutes(api, fiberAdapter.NewAnalyticsHandler(analyticsSvc, log))
}

func setupMarket(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, mcpServer *mcpAdapter.Server, cacheClient port.CacheClient) {
	providers := make([]port.MarketProvider, 0, 2)
	fmarket, err := fmarketProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("FMARKET_BASE_URL", "https://api.fmarket.vn"), log)
	if err != nil {
		log.LogError(ctx, "failed to initialize market provider fmarket", err)
		fmarket = nil
	} else {
		providers = append(providers, fmarket)
	}
	vnstock, err := vnstockProvider.NewProvider(configAdapter.LoadMarketDataProviderConfig("VNSTOCK_SERVER_URL", "http://localhost:8000"), log)
	if err != nil {
		log.LogError(ctx, "failed to initialize market provider vnstock", err)
	} else {
		providers = append(providers, vnstock)
	}
	for _, provider := range providers {
		health.Register(port.HealthCheck{Name: "market:" + provider.Name(), Probe: func(ctx context.Context) error {
			_, err := provider.Health(ctx)
			return err
		}})
	}
	if len(providers) == 0 {
		return
	}
	marketSvc := marketProviderService.NewServiceWithRouting(domain.DefaultMarketRoutingConfig(), cacheClient, log, providers...)
	mcpServer.SetMarketService(marketSvc)
	fiberAdapter.RegisterMarketRoutes(api, fiberAdapter.NewMarketHandler(marketSvc, log), fiberAdapter.NewMarketProviderHandler(marketSvc, log))
	if fmarket != nil {
		fmarketSvc := fmarketProvider.NewService(fmarket, cacheClient)
		mcpServer.SetFmarketService(fmarketSvc)
		fiberAdapter.RegisterFmarketRoutes(api, fiberAdapter.NewFmarketHandler(fmarketSvc, marketSvc, log))
	}
}
