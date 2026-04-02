package main

import (
	cacheAdapter "apps/wealth-management-engine/adapter/cache"
	configAdapter "apps/wealth-management-engine/adapter/config"
	dbAdapter "apps/wealth-management-engine/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management-engine/adapter/fiber"
	loggerAdapter "apps/wealth-management-engine/adapter/logger"
	fmarketProvider "apps/wealth-management-engine/adapter/market/fmarket"
	vnstockProvider "apps/wealth-management-engine/adapter/market/vnstock"
	mcpAdapter "apps/wealth-management-engine/adapter/mcp"
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"apps/wealth-management-engine/service"
	cacheService "apps/wealth-management-engine/service/cache"
	marketProviderService "apps/wealth-management-engine/service/market_provider"
	"context"
	"github.com/gofiber/fiber/v2"
	"os"
)

func main() {
	// Initialize logger with environment settings
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	colorEnabled := true
	log := loggerAdapter.NewLogger(logLevel, colorEnabled)

	healthSvc := service.NewHealthService()
	mcpServer := mcpAdapter.NewServer(healthSvc, nil, nil, nil, log)

	configAdapter.Load()

	app := fiber.New()
	ctx := context.Background()

	// Add middleware
	app.Use(fiberAdapter.RequestIDMiddleware(log))
	app.Use(fiberAdapter.LoggingMiddleware(log))
	app.Use(fiberAdapter.RecoveryMiddleware(log))

	// Initialize services
	sheetsConfig, err := configAdapter.LoadSheetsConfig()
	if err != nil {
		log.LogError(ctx, "failed to load google sheets config", err)
	}
	cacheConfig, cacheConfigErr := configAdapter.LoadCacheConfig()
	if cacheConfigErr != nil {
		log.LogError(ctx, "failed to load cache config", cacheConfigErr)
	}
	fmarketProviderConfig := configAdapter.LoadMarketDataProviderConfig("FMARKET_BASE_URL", "https://api.fmarket.vn")
	vnstockProviderConfig := configAdapter.LoadMarketDataProviderConfig("VNSTOCK_SERVER_URL", "http://localhost:8000")
	var cacheClient port.CacheClient
	var cacheSvc port.CacheService
	var dbSvc port.DatabaseService
	var fmarketSvc port.FmarketService
	var marketSvc port.MarketProviderService

	// Initialize handlers
	healthHandler := fiberAdapter.NewHealthHandler(healthSvc)
	openAPIHandler := fiberAdapter.NewOpenAPIHandler(app)
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")
	api.Get("/health", healthHandler.HealthCheck, fiberAdapter.RouteMeta{Summary: "Checks platform liveness and backend service connectivity. Use to verify that the backend and all dependencies are healthy. Returns system status and relevant diagnostics.", Tags: []string{"health"}})
	app.Get("/api/openapi.json", openAPIHandler.Spec)
	app.Get("/api/docs", openAPIHandler.Docs)

	if cacheConfigErr == nil {
		createdCacheClient, cacheClientErr := cacheAdapter.NewClient(cacheConfig, log)
		if cacheClientErr != nil {
			log.LogError(ctx, "failed to initialize upstash redis client", cacheClientErr)
		} else {
			cacheClient = createdCacheClient
			cacheSvc = cacheService.New(cacheClient, log)
			cacheHandler := fiberAdapter.NewCacheHandler(cacheSvc, log)
			cache := api.Group("/cache", "cache")
			cache.Get("/health", cacheHandler.Ping, fiberAdapter.RouteMeta{Summary: "Checks the online status and availability of the configured caching backend (such as Redis). Use to confirm cache layer connectivity and readiness.", Tags: []string{"cache"}})
			cache.Post("/set", cacheHandler.Set, fiberAdapter.RouteMeta{Summary: "Sets an arbitrary cache entry. Call with a key/value payload to create or update values in the backend cache for accelerated lookups by other API flows.", Tags: []string{"cache"}})
			cache.Get("/get/:key", cacheHandler.Get, fiberAdapter.RouteMeta{
				Summary: "Retrieves the value (if present) for a specific cache key. Used for debugging, admin tools, or custom caching of frequently accessed data elements.",
				Tags:    []string{"cache"},
				PathParams: []fiberAdapter.RouteParameter{
					fiberAdapter.PathParam("key", "string", "Cache entry key."),
				},
			})
			cache.Delete("/invalidate", cacheHandler.Invalidate, fiberAdapter.RouteMeta{
				Summary: "Deletes (invalidates) one or more keys in the backend cache. Useful after data changes or if stale data is suspected. Use the `pattern` query to specify affected keys.",
				Tags:    []string{"cache"},
				QueryParams: []fiberAdapter.RouteParameter{
					fiberAdapter.QueryParam("pattern", "string", "Cache key pattern to invalidate.", "*"),
				},
			})
		}
	}

	if err == nil {
		dbClient, clientErr := dbAdapter.NewSheetsClient(ctx, sheetsConfig, log)
		if clientErr != nil {
			log.LogError(ctx, "failed to initialize google sheets client", clientErr)
		} else {
			dbSvc = service.NewDatabaseService(dbClient, cacheSvc, log)
			mcpServer.SetDatabaseService(dbSvc)
			dbHandler := fiberAdapter.NewDatabaseHandler(dbSvc, log)
			sheets := api.Group("/sheets", "sheets")
			sheets.Get("/accounts", dbHandler.GetSheetsAccounts, fiberAdapter.RouteMeta{Summary: "Returns a raw export of account rows directly from the connected Google Sheets. Use for diagnostic visibility and troubleshooting synchronization and data mapping.", Tags: []string{"sheets"}})

			accounts := api.Group("/accounts", "accounts")
			accounts.Get("/", dbHandler.GetAccounts, fiberAdapter.RouteMeta{
				Summary: "Fetches the user's current accounts and their latest details as known to backend (name, metadata, balances). Use `?force=true` to bypass cache and fetch from Google Sheets directly.",
				Tags:    []string{"accounts"},
				QueryParams: []fiberAdapter.RouteParameter{
					fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false),
				},
			})

			transactions := api.Group("/transactions", "transactions")
			transactions.Get("/", dbHandler.GetTransactions, fiberAdapter.RouteMeta{
				Summary: "Returns all transactions associated with linked accounts, including metadata columns. Use `?force=true` to bypass cache and fetch directly.",
				Tags:    []string{"transactions"},
				QueryParams: []fiberAdapter.RouteParameter{
					fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false),
				},
			})
			transactions.Post("/", dbHandler.CreateTransaction, fiberAdapter.RouteMeta{
				Summary: "Creates a new transaction in the active budget sheet. The request must include: accountName, date, payee, category. Optional fields: tags, cleared, payment, deposit, memo, referenceNumber. On success, persists and returns transaction row details.",
				Tags:    []string{"transactions"},
				RequestBody: map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"accountName":     map[string]any{"type": "string"},
									"date":            map[string]any{"type": "string"},
									"payee":           map[string]any{"type": "string"},
									"category":        map[string]any{"type": "string"},
									"tags":            map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
									"cleared":         map[string]any{"type": "boolean"},
									"payment":         map[string]any{"type": "number"},
									"deposit":         map[string]any{"type": "number"},
									"memo":            map[string]any{"type": "string"},
									"referenceNumber": map[string]any{"type": "string"},
								},
								"required": []string{"accountName", "date", "payee", "category"},
							},
						},
					},
				},
			})

			budget := api.Group("/budget", "budget")
			budget.Get("/", dbHandler.GetBudget, fiberAdapter.RouteMeta{Summary: "Fetches the current budget summary, including allocations and spending to date. Use `?force=true` to request fresh data from Google Sheets.", Tags: []string{"budget"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			categories := api.Group("/categories", "categories")
			categories.Get("/", dbHandler.GetCategories, fiberAdapter.RouteMeta{Summary: "Lists all categories used in transaction classification and budget planning. Use to power forms and give UI context. Use `?force=true` to bypass cache.", Tags: []string{"categories"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			goals := api.Group("/goals", "goals")
			goals.Get("/", dbHandler.GetGoals, fiberAdapter.RouteMeta{Summary: "Lists all financial goals tracked in the main sheet (name, amount, progress, etc). Use `?force=true` to bypass cache.", Tags: []string{"goals"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			loans := api.Group("/loans", "loans")
			loans.Get("/", dbHandler.GetLoans, fiberAdapter.RouteMeta{Summary: "Lists all known loans, their key attributes, and payoff progress. Use for dashboards, analysis, or budgeting. `?force=true` for fresh sheet read.", Tags: []string{"loans"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			notifications := api.Group("/notifications", "notifications")
			notifications.Get("/", dbHandler.GetNotifications, fiberAdapter.RouteMeta{Summary: "Fetches current actionable notifications (e.g., required updates, anomalies) for this user or sheet scope. Use `?force=true` to get fresh notices.", Tags: []string{"notifications"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})
			notifications.Patch("/", dbHandler.MarkNotificationsDone, fiberAdapter.RouteMeta{
				Summary: "Marks one or more notifications as completed (requires array of row numbers in request body). Used to mute or clear handled notifications.",
				Tags:    []string{"notifications"},
				RequestBody: map[string]any{
					"required": true,
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"rowNumbers": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}},
								},
								"required": []string{"rowNumbers"},
							},
						},
					},
				},
			})

			tags := api.Group("/tags", "tags")
			tags.Get("/", dbHandler.GetTags, fiberAdapter.RouteMeta{Summary: "Fetches all custom tags defined for categorizing transactions. Use for autocomplete and UIs. `?force=true` for live sheet read.", Tags: []string{"tags"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			sync := api.Group("/sync", "sync")
			sync.Post("/", dbHandler.Sync, fiberAdapter.RouteMeta{Summary: "Clears and invalidates all major cached app data blocks (accounts, transactions, budgets, categories, goals, notifications, investments, and AI content). Forces a full cache reset, making the backend reload fresh information from its sources (e.g., Google Sheets). Useful after batch updates, configuration changes, or to solve data hygiene issues.", Tags: []string{"sync"}})

			init := api.Group("/init", "init")
			init.Get("/", dbHandler.Init, fiberAdapter.RouteMeta{Summary: "Initializes and verifies readiness of AI-related content (prompts and knowledge) from Google Sheets. Triggers warmup/preload of this content into cache, ensuring all backend AI features are prepared. Returns counts of loaded prompts/knowledge rows. Use ?force=true to bypass cache and perform a live data fetch.", Tags: []string{"init"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})

			investments := api.Group("/investments", "investments")
			investments.Get("/assets", dbHandler.GetInvestmentAssets, fiberAdapter.RouteMeta{Summary: "Retrieves details of investment assets (e.g., shares, funds) tracked in the user's investment sheet, including type, balance, and last update. Use for dashboarding and portfolio analysis. `?force=true` for live sheet read.", Tags: []string{"investments"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("force", "boolean", "Bypass cache and fetch fresh data from Google Sheets.", false)}})
		}
	}

	marketProviders := make([]port.MarketProvider, 0, 2)
	fmarket, fmarketErr := fmarketProvider.NewProvider(fmarketProviderConfig, log)
	if fmarketErr != nil {
		log.LogError(ctx, "failed to initialize market provider fmarket", fmarketErr)
	} else {
		marketProviders = append(marketProviders, fmarket)
	}
	vnstock, vnstockErr := vnstockProvider.NewProvider(vnstockProviderConfig, log)
	if vnstockErr != nil {
		log.LogError(ctx, "failed to initialize market provider vnstock", vnstockErr)
	} else {
		marketProviders = append(marketProviders, vnstock)
	}
	if len(marketProviders) > 0 {
		marketRouting := domain.DefaultMarketRoutingConfig()
		marketSvc = marketProviderService.NewServiceWithRouting(marketRouting, cacheClient, log, marketProviders...)
		mcpServer.SetMarketService(marketSvc)
		marketProviderHandler := fiberAdapter.NewMarketProviderHandler(marketSvc, log)
		marketHandler := fiberAdapter.NewMarketHandler(marketSvc, log)
		external := api.Group("/external", "external")
		marketAPI := api.Group("/market", "market")
		if fmarket != nil {
			fmarketSvc = fmarketProvider.NewService(fmarket, cacheClient)
			mcpServer.SetFmarketService(fmarketSvc)
		}

		externalMarket := external.Group("/market", "market")
		externalMarket.Get("/providers/:provider/health", marketProviderHandler.Health, fiberAdapter.RouteMeta{Summary: "Checks connectivity with a specified external market data provider (e.g., Fmarket, vnstock). Useful to verify third-party integrations are online.", Tags: []string{"market"}, PathParams: []fiberAdapter.RouteParameter{fiberAdapter.PathParam("provider", "string", "Market provider name.")}})
		external.Get("/vnstock/health", marketProviderHandler.Health, fiberAdapter.RouteMeta{Summary: "Checks the health and network status of the vnstock market provider. For integration validation and diagnostics.", Tags: []string{"vnstock"}})

		marketAPI.Get("/ticker", marketHandler.GetTicker, fiberAdapter.RouteMeta{Summary: "Returns current ticker price and info for a symbol. Params: `symbol`, `type` (ticker type), and optionally `skipCache=true`. Use for instant price lookups or graphs.", Tags: []string{"market"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("skipCache", "boolean", "Bypass market cache and fetch fresh provider data.", false), fiberAdapter.QueryParam("symbol", "string", "Ticker symbol to fetch."), fiberAdapter.QueryParam("type", "string", "Ticker type.")}})
		marketAPI.Get("/exchange-rate", marketHandler.GetExchangeRate, fiberAdapter.RouteMeta{Summary: "Returns the latest exchange rate between two currency codes. Requires `from`, `to` parameters. Optionally, `skipCache=true` for a fresh quote.", Tags: []string{"market"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("skipCache", "boolean", "Bypass market cache and fetch fresh provider data.", false), fiberAdapter.QueryParam("from", "string", "Source currency code."), fiberAdapter.QueryParam("to", "string", "Target currency code.")}})
		marketAPI.Get("/price-series", marketHandler.GetPriceSeries, fiberAdapter.RouteMeta{Summary: "Returns a series of historic price points for a symbol and type. Supports charting/analysis. `skipCache`, `symbol`, `type` are query parameters.", Tags: []string{"market"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("skipCache", "boolean", "Bypass market cache and fetch fresh provider data.", false), fiberAdapter.QueryParam("symbol", "string", "Ticker symbol to fetch."), fiberAdapter.QueryParam("type", "string", "Series type.")}})
		marketAPI.Get("/bank-rates", marketHandler.GetBankInterestRate, fiberAdapter.RouteMeta{Summary: "Returns current published savings and lending rates for supported banks. Optionally, pass `skipCache` for live data.", Tags: []string{"market"}, QueryParams: []fiberAdapter.RouteParameter{fiberAdapter.QueryParam("skipCache", "boolean", "Bypass market cache and fetch fresh provider data.", false)}})
	}

	go func() {
		if err := mcpServer.Run(os.Stdin, os.Stdout); err != nil {
			log.LogError(ctx, "mcp server stopped", err)
		}
	}()

	log.LogApplicationEvent(ctx, "starting server on :8080")
	if err := app.Listen(":8080"); err != nil {
		log.LogError(ctx, "server failed to start", err)
		os.Exit(1)
	}
}
