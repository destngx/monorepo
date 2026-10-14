package main

import (
	openaiAdapter "apps/wealth-management/adapter/ai/openai"
	cacheAdapter "apps/wealth-management/adapter/cache"
	configAdapter "apps/wealth-management/adapter/config"
	dbAdapter "apps/wealth-management/adapter/db/google_sheets"
	fiberAdapter "apps/wealth-management/adapter/fiber"
	googleauthAdapter "apps/wealth-management/adapter/googleauth"
	loggerAdapter "apps/wealth-management/adapter/logger"
	marketData "apps/wealth-management/adapter/market/marketdata"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	duckduckgoSearch "apps/wealth-management/adapter/search/duckduckgo"
	exaSearch "apps/wealth-management/adapter/search/exa"
	tavilySearch "apps/wealth-management/adapter/search/tavily"
	settingsAdapter "apps/wealth-management/adapter/settings"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"apps/wealth-management/service"
	aiService "apps/wealth-management/service/ai"
	cacheService "apps/wealth-management/service/cache"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

func main() {
	os.Exit(run())
}

// run serves until the server fails (exit 1) or a settings restart is
// requested, in which case the process replaces itself with a fresh copy of
// the binary so every service is rebuilt from the saved settings.
func run() int {
	ctx := context.Background()
	colorEnabled := configAdapter.LoadLogColor()
	serverConfig, err := configAdapter.LoadServerConfig()
	if err != nil {
		loggerAdapter.NewLogger("info", colorEnabled).LogError(ctx, "invalid server configuration", err)
		return 1
	}
	store, values, err := openSettings()
	if err != nil {
		loggerAdapter.NewLogger("info", colorEnabled).LogError(ctx, "cannot open the settings store", err)
		return 1
	}
	defer store.Close()
	log := loggerAdapter.NewLogger(configAdapter.LoadLogLevel(values), colorEnabled)
	log.LogApplicationEvent(ctx, "settings loaded from "+store.Path())

	// A bad stored value must not stop the engine, or the Settings page
	// could not be reached to fix it: fall back to the defaults.
	cardConfig, err := configAdapter.LoadCardConfig(values)
	if err != nil {
		log.LogError(ctx, "invalid card programs configuration: using the built-in programs", err)
		if cardConfig, err = configAdapter.LoadCardConfig(configAdapter.Values{}); err != nil {
			log.LogError(ctx, "invalid built-in card programs", err)
			return 1
		}
	}
	captureLimits, err := configAdapter.LoadCaptureLimits(values)
	if err != nil {
		log.LogError(ctx, "invalid Smart Capture configuration: using the default limits", err)
		captureLimits = domain.DefaultCaptureLimits()
	}

	health := service.NewHealthService()
	mcpServer := mcpAdapter.NewServer(health, log)
	mcpServer.SetMaxRequestBytes(bodyLimit(captureLimits))

	// No startup banner: stdout carries the MCP stdio stream. Immutable keeps
	// request strings valid after a handler returns (they reach caches and logs).
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Immutable:             true,
		BodyLimit:             bodyLimit(captureLimits),
		ErrorHandler:          fiberAdapter.FiberErrorHandler(log),
	})
	app.Use(fiberAdapter.RequestIDMiddleware(log))
	app.Use(fiberAdapter.LoggingMiddleware(log))
	app.Use(fiberAdapter.RecoveryMiddleware(log))
	staleData := fiberAdapter.NewStaleDataMiddleware()
	app.Use(staleData.Handler())

	guard := fiberAdapter.NewWriteGuard(serverConfig.APIToken, serverConfig.Host)
	if serverConfig.APIToken == "" {
		log.LogApplicationEvent(ctx, "API_TOKEN is not set: write endpoints only work when HOST is a loopback address")
	}
	openAPIHandler := fiberAdapter.NewOpenAPIHandler(app)
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")
	api.Get("/health", fiberAdapter.NewHealthHandler(health).HealthCheck, fiberAdapter.RouteMeta{Summary: "Reports the status of each dependency (sheets, cache, market providers). 503 when any is down or unconfigured.", Tags: []string{"health"}})
	app.Get("/api/openapi.json", openAPIHandler.Spec)
	app.Get("/api/docs", openAPIHandler.Docs)

	restart := make(chan struct{}, 1)
	settingsSvc := service.NewSettingsService(store, func() {
		select {
		case restart <- struct{}{}:
		default:
		}
	})
	fiberAdapter.RegisterSettingsRoutes(api, fiberAdapter.NewSettingsHandler(settingsSvc, log), guard)
	googleAuthSvc := service.NewGoogleAuthService(store, settingsSvc, googleauthAdapter.New(), defaultGoogleRedirect(serverConfig.Port))
	fiberAdapter.RegisterGoogleAuthRoutes(api, fiberAdapter.NewGoogleAuthHandler(googleAuthSvc, log), guard)

	cacheClient, cacheSvc := setupCache(ctx, log, health, api, values)
	if closer, ok := cacheClient.(io.Closer); ok {
		defer closer.Close()
	}
	dbSvc, analyticsSvc := setupSheets(ctx, log, health, api, guard, mcpServer, cacheSvc, cardConfig, values)
	if dbSvc != nil {
		staleData.Watch(dbSvc)
	}
	marketSvc := setupMarket(ctx, log, health, api, mcpServer, values)
	aiOptions := aiService.Options{CaptureLimits: captureLimits, Search: setupSearch(ctx, log, cacheSvc, values)}
	if marketSvc != nil {
		analysisSvc := service.NewMarketAnalysisService(marketSvc, nil)
		fiberAdapter.RegisterMarketAnalysisRoutes(api, fiberAdapter.NewMarketAnalysisHandler(analysisSvc, log))
		aiOptions.Market = analysisSvc
		mcpServer.SetMarketAnalysisService(analysisSvc)
	}
	if dbSvc != nil && marketSvc != nil {
		portfolioSvc := service.NewPortfolioService(dbSvc, marketSvc, nil)
		fiberAdapter.RegisterPortfolioRoutes(api, fiberAdapter.NewPortfolioHandler(portfolioSvc, log))
		aiOptions.Portfolio = portfolioSvc
		mcpServer.SetPortfolioService(portfolioSvc)
	}
	setupAI(ctx, log, health, api, guard, mcpServer, dbSvc, analyticsSvc, cacheSvc, aiOptions, values)

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
	served := make(chan error, 1)
	go func() { served <- app.Listen(address) }()

	select {
	case err := <-served:
		log.LogError(ctx, "server failed to start", err)
		return 1
	case <-restart:
	}
	// Shutdown lets in-flight requests finish, so the restart request gets
	// its 202 before the process is replaced.
	log.LogApplicationEvent(ctx, "restarting to apply saved settings")
	if err := app.ShutdownWithTimeout(restartShutdownTimeout); err != nil {
		log.LogError(ctx, "graceful shutdown before restart failed", err)
	}
	if err := <-served; err != nil {
		log.LogError(ctx, "server stopped with an error before restart", err)
	}
	if closer, ok := cacheClient.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			log.LogError(ctx, "closing the cache before restart failed", err)
		}
	}
	if err := store.Close(); err != nil {
		log.LogError(ctx, "closing the settings store before restart failed", err)
	}
	err = reexec()
	log.LogError(ctx, "restart failed: start the engine again by hand", err)
	return 1
}

const restartShutdownTimeout = 10 * time.Second

// reexec replaces the process with the same binary, arguments and
// environment, keeping the pid (so air and go run keep tracking it) and the
// stdio streams (MCP clients reconnect on their side).
func reexec() error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	return syscall.Exec(executable, os.Args, os.Environ())
}

// defaultGoogleRedirect is the engine's own Google sign-in callback on
// loopback, which a Desktop OAuth client accepts on any port. A browser on
// another machine cannot reach it; GOOGLE_REDIRECT_URL or the paste fallback
// covers that.
func defaultGoogleRedirect(port int) string {
	return "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + domain.GoogleAuthCallbackPath
}

// openSettings opens WM_SETTINGS_PATH, or settings.db in the user config
// directory, and reads every stored setting.
func openSettings() (*settingsAdapter.Store, configAdapter.Values, error) {
	path, err := settingsAdapter.ResolvePath(configAdapter.LoadSettingsPath())
	if err != nil {
		return nil, nil, err
	}
	store, err := settingsAdapter.Open(path)
	if err != nil {
		return nil, nil, err
	}
	values, err := store.Values(context.Background())
	if err != nil {
		return nil, nil, errors.Join(err, store.Close())
	}
	return store, configAdapter.Values(values), nil
}

func setupCache(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, values configAdapter.Values) (port.CacheClient, port.CacheService) {
	cacheConfig, err := configAdapter.LoadCacheConfig(values)
	if err != nil {
		log.LogError(ctx, "cache disabled: reads go straight to the sheet", err)
		health.Register(port.HealthCheck{Name: "cache"})
		return nil, nil
	}
	client, err := cacheAdapter.NewBackend(cacheConfig, log)
	if err != nil {
		log.LogError(ctx, "failed to initialize the "+string(cacheConfig.Backend)+" cache: reads go straight to the sheet", err)
		health.Register(port.HealthCheck{Name: "cache"})
		return nil, nil
	}
	cacheSvc := cacheService.New(client, log)
	health.Register(port.HealthCheck{Name: "cache", Probe: cacheSvc.Ping})
	api.Group("/cache", "cache").Get("/health", fiberAdapter.NewCacheHandler(cacheSvc, log).Ping, fiberAdapter.RouteMeta{Summary: "Checks the cache backend."})
	return client, cacheSvc
}

// setupSheets returns nil services when the sheet is not configured; its
// routes are still registered and answer 503 SHEETS_NOT_CONFIGURED naming
// the missing settings (or the client error) so clients can say what to fix.
func setupSheets(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, guard fiberAdapter.WriteGuard, mcpServer *mcpAdapter.Server, cacheSvc port.CacheService, cards domain.CardConfig, values configAdapter.Values) (port.DatabaseService, port.AnalyticsService) {
	sheetsConfig, err := configAdapter.LoadSheetsConfig(values)
	if err != nil {
		log.LogError(ctx, "sheet features disabled", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		fiberAdapter.RegisterUnconfiguredSheetRoutes(api, err, log)
		return nil, nil
	}
	client, err := dbAdapter.NewSheetsClient(ctx, sheetsConfig, log)
	if err != nil {
		log.LogError(ctx, "failed to initialize google sheets client", err)
		health.Register(port.HealthCheck{Name: "sheets"})
		fiberAdapter.RegisterUnconfiguredSheetRoutes(api, fmt.Errorf("%w: google sheets client could not be built: %w", configAdapter.ErrMissingSheetsConfig, err), log)
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
	dbSvc := service.NewDatabaseService(client, cacheSvc, log, service.DatabaseServiceOptions{
		BudgetSheetTemplate: configAdapter.BudgetSheetTemplate(values),
		SpreadsheetID:       sheetsConfig.SpreadsheetID,
	})
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
func setupAI(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, guard fiberAdapter.WriteGuard, mcpServer *mcpAdapter.Server, dbSvc port.DatabaseService, analyticsSvc port.AnalyticsService, cacheSvc port.CacheService, options aiService.Options, values configAdapter.Values) {
	if dbSvc == nil {
		log.LogApplicationEvent(ctx, "AI features disabled: they need the Google Sheet")
		return
	}
	aiConfig, err := configAdapter.LoadAIConfig(values)
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
func setupSearch(ctx context.Context, log *loggerAdapter.Logger, cacheSvc port.CacheService, values configAdapter.Values) port.SearchService {
	config := configAdapter.LoadSearchConfig(values)
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

// setupMarket wires the market data server client (WM-MKT-1): the server
// owns every source, provider chain, cache and last-good copy, so the
// engine keeps no market layer of its own. It returns nil when
// MARKET_DATA_SERVER_URL is invalid.
func setupMarket(ctx context.Context, log *loggerAdapter.Logger, health service.HealthRegistry, api fiberAdapter.RouteRegistrar, mcpServer *mcpAdapter.Server, values configAdapter.Values) port.MarketProviderService {
	client, err := marketData.NewClient(configAdapter.LoadMarketDataServerURL(values))
	if err != nil {
		log.LogError(ctx, "market data disabled", err)
		return nil
	}
	log.LogApplicationEvent(ctx, "market data server: "+client.BaseURL())
	health.Register(port.HealthCheck{Name: "market-data-server", Probe: client.Ping})
	mcpServer.SetMarketService(client)
	mcpServer.SetFmarketService(client)
	fiberAdapter.RegisterMarketRoutes(api, fiberAdapter.NewMarketHandler(client, log), fiberAdapter.NewMarketProviderHandler(client, log))
	fiberAdapter.RegisterFmarketRoutes(api, fiberAdapter.NewFmarketHandler(client, client, log))
	return client
}
