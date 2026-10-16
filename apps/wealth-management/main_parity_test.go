package main

import (
	fiberAdapter "apps/wealth-management/adapter/fiber"
	loggerAdapter "apps/wealth-management/adapter/logger"
	mcpAdapter "apps/wealth-management/adapter/mcp"
	"apps/wealth-management/internal/fakes"
	"apps/wealth-management/service"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// routeTools maps every HTTP capability to its MCP tool (WM-MCP-4).
var routeTools = map[string]string{
	"GET /api/health": "EngineHealth",

	"GET /api/accounts":                 "GetAccounts",
	"POST /api/accounts":                "CreateAccount",
	"GET /api/transactions":             "GetTransactions",
	"POST /api/transactions":            "CreateTransaction",
	"POST /api/transactions/batch":      "CreateTransactions",
	"GET /api/budget":                   "GetBudget",
	"POST /api/budget/reallocate":       "ReallocateBudget",
	"GET /api/sheets/info":              "GetSheetInfo",
	"GET /api/analytics/budget/history": "GetBudgetHistory",
	"GET /api/categories":               "GetCategories",
	"GET /api/goals":                    "GetGoals",
	"GET /api/loans":                    "GetLoans",
	"GET /api/tags":                     "GetTags",
	"GET /api/investments/assets":       "GetInvestmentAssets",
	"GET /api/capture/rules":            "GetCaptureRules",
	"POST /api/capture/rules":           "SaveCaptureRules",
	"GET /api/cards/programs":           "GetCardPrograms",
	"POST /api/sync":                    "SyncSheetsCache",
	"GET /api/init":                     "WarmAIContent",

	"GET /api/analytics/dashboard":         "GetDashboard",
	"GET /api/analytics/budget":            "GetBudgetReport",
	"GET /api/analytics/cards":             "GetCardReports",
	"GET /api/analytics/cards/suggest":     "SuggestCardForPayment",
	"GET /api/analytics/cards/compare":     "CompareCardPrograms",
	"GET /api/analytics/health":            "GetFinancialHealth",
	"GET /api/analytics/account-histories": "GetAccountHistories",
	"GET /api/analytics/cash-flow/yearly":  "GetYearlyCashFlow",

	"GET /api/external/market/providers/:provider/health": "GetMarketProviderHealth",
	"GET /api/external/vnstock/health":                    "GetMarketProviderHealth",
	"GET /api/fmarket/health":                             "GetMarketProviderHealth",
	"POST /api/fmarket/action":                            "RunFmarketAction",
	"GET /api/market/ticker":                              "GetMarketTicker",
	"GET /api/market/exchange-rate":                       "GetExchangeRate",
	"GET /api/market/price-series":                        "GetPriceSeries",
	"GET /api/market/candles":                             "GetMarketCandles",
	"GET /api/market/bank-rates":                          "GetBankRates",
	"GET /api/market/pulse":                               "GetMarketPulse",
	"GET /api/market/analysis":                            "GetMarketAnalysis",
	"GET /api/investments/valuation":                      "GetInvestmentValuation",

	"GET /api/ai/status":               "GetAIStatus",
	"GET /api/ai/usage":                "GetAIUsage",
	"GET /api/ai/models":               "GetAIModels",
	"POST /api/ai/suggest-category":    "SuggestCategory",
	"POST /api/ai/review":              "ReviewFinances",
	"POST /api/ai/health-commentary":   "GetHealthCommentary",
	"POST /api/ai/chart-insight":       "GetChartInsight",
	"POST /api/ai/cashback-explain":    "ExplainCashback",
	"POST /api/ai/briefing":            "GetIntelligenceBriefing",
	"POST /api/ai/budget-advisor":      "GetBudgetAdvice",
	"POST /api/ai/capture":             "SuggestTransactionsFromMaterials",
	"POST /api/ai/chat":                "Chat",
	"POST /api/ai/news-sentiment":      "GetNewsSentiment",
	"POST /api/ai/ticker-insight":      "GetTickerInsight",
	"POST /api/ai/investment-analysis": "RunInvestmentAnalysis",
}

// unexposedRoutes are HTTP routes intentionally without an MCP tool.
var unexposedRoutes = map[string]string{
	"GET /api/openapi.json":          "describes the HTTP API itself; MCP clients get tools/list instead",
	"GET /api/docs":                  "Swagger UI HTML for browsers, not data",
	"GET /api/cache/health":          "cache backend ping; EngineHealth already reports the cache check",
	"GET /api/settings":              "engine configuration and credentials are edited by the user in the UI, not by agents",
	"POST /api/settings/restart":     "restarts the engine, and with it the MCP session an agent would call it from",
	"POST /api/auth/google/start":    "connecting a Google account needs the user's consent in a browser",
	"GET /api/auth/google/callback":  "Google redirects the user's browser here after consent",
	"POST /api/auth/google/exchange": "completes the user's browser consent; an agent has no consent to complete",
}

// TestGivenEveryHTTPRouteWhenListingMCPToolsThenEachIsMappedOrAllowlisted
// enforces the parity rule: a new GET or POST route fails this test until
// it gets a tool or a justified allowlist entry.
func TestGivenEveryHTTPRouteWhenListingMCPToolsThenEachIsMappedOrAllowlisted(t *testing.T) {
	routes := registeredRoutes(t)
	tools := map[string]bool{}
	for _, definition := range fullMCPServer(t).ListTools() {
		tools[definition.Name] = true
	}

	for _, route := range routes {
		tool, mapped := routeTools[route]
		_, allowlisted := unexposedRoutes[route]
		switch {
		case mapped && allowlisted:
			t.Errorf("%s is both mapped to %s and allowlisted", route, tool)
		case mapped && !tools[tool]:
			t.Errorf("%s maps to %s, which the MCP server does not list", route, tool)
		case !mapped && !allowlisted:
			t.Errorf("%s has no MCP tool: add one (WM-MCP-4) or allowlist it with a reason", route)
		}
	}
	for route := range routeTools {
		if !slices.Contains(routes, route) {
			t.Errorf("routeTools lists %s, which is no longer registered", route)
		}
	}
	for route := range unexposedRoutes {
		if !slices.Contains(routes, route) {
			t.Errorf("unexposedRoutes lists %s, which is no longer registered", route)
		}
	}
}

// registeredRoutes registers every route group the way main does and
// returns the GET and POST routes as "METHOD /path". Handlers are never
// invoked, so they get no services. The UI is mounted outside /api and is
// not an API capability.
func registeredRoutes(t *testing.T) []string {
	t.Helper()
	log := loggerAdapter.NewTestLogger(t)
	app := fiber.New()
	guard := fiberAdapter.NewWriteGuard("", "127.0.0.1")
	api := fiberAdapter.NewRouteRegistrar(app, "/api", "")
	openAPI := fiberAdapter.NewOpenAPIHandler(app)

	api.Get("/health", fiberAdapter.NewHealthHandler(service.NewHealthService()).HealthCheck, fiberAdapter.RouteMeta{Summary: "health"})
	app.Get("/api/openapi.json", openAPI.Spec)
	app.Get("/api/docs", openAPI.Docs)
	api.Group("/cache", "cache").Get("/health", fiberAdapter.NewCacheHandler(nil, log).Ping, fiberAdapter.RouteMeta{Summary: "cache"})
	fiberAdapter.RegisterSheetRoutes(api, fiberAdapter.NewDatabaseHandler(nil, log), guard)
	fiberAdapter.RegisterSettingsRoutes(api, fiberAdapter.NewSettingsHandler(nil, log), guard)
	fiberAdapter.RegisterGoogleAuthRoutes(api, fiberAdapter.NewGoogleAuthHandler(nil, log), guard)
	fiberAdapter.RegisterAnalyticsRoutes(api, fiberAdapter.NewAnalyticsHandler(nil, log))
	fiberAdapter.RegisterMarketRoutes(api, fiberAdapter.NewMarketHandler(nil, log), fiberAdapter.NewMarketProviderHandler(nil, log))
	fiberAdapter.RegisterFmarketRoutes(api, fiberAdapter.NewFmarketHandler(nil, nil, log))
	fiberAdapter.RegisterMarketAnalysisRoutes(api, fiberAdapter.NewMarketAnalysisHandler(nil, log))
	fiberAdapter.RegisterPortfolioRoutes(api, fiberAdapter.NewPortfolioHandler(nil, log))
	fiberAdapter.RegisterAIRoutes(api, fiberAdapter.NewAIHandler(nil, log), guard)

	seen := map[string]bool{}
	routes := []string{}
	for _, route := range app.GetRoutes(true) {
		if route.Method != http.MethodGet && route.Method != http.MethodPost {
			continue
		}
		path := route.Path
		if len(path) > 1 {
			path = strings.TrimSuffix(path, "/")
		}
		key := route.Method + " " + path
		if !seen[key] {
			seen[key] = true
			routes = append(routes, key)
		}
	}
	slices.Sort(routes)
	return routes
}

func fullMCPServer(t *testing.T) *mcpAdapter.Server {
	t.Helper()
	server := mcpAdapter.NewServer(nil, loggerAdapter.NewTestLogger(t))
	market := &fakes.Market{}
	server.SetDatabaseService(&fakes.Database{})
	server.SetAnalyticsService(service.NewAnalyticsService(&fakes.Database{}, nil))
	server.SetMarketService(market)
	server.SetFmarketService(market)
	server.SetMarketAnalysisService(market)
	server.SetPortfolioService(market)
	server.SetAIService(&fakes.AI{})
	return server
}
