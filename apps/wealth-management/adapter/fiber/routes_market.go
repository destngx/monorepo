package fiber

import "apps/wealth-management/domain"

func skipCacheParam() RouteParameter {
	return QueryParam("skipCache", "boolean", "Bypass the market cache and fetch fresh provider data.", false)
}

func RegisterMarketRoutes(api RouteRegistrar, market *MarketHandler, providers *MarketProviderHandler) {
	external := api.Group("/external", "external")
	external.Group("/market", "market").Get("/providers/:provider/health", providers.Health, RouteMeta{Summary: "Checks connectivity with a market data provider .", Tags: []string{"market"}, PathParams: []RouteParameter{PathParam("provider", "string", "Market provider name (fmarket, vnstock, coingecko, currencyapi, yahoo).")}})
	external.Get("/vnstock/health", providers.Health, RouteMeta{Summary: "Checks the vnstock provider.", Tags: []string{"vnstock"}})

	marketAPI := api.Group("/market", "market")
	marketAPI.Get("/ticker", market.GetTicker, RouteMeta{Summary: "Current quote for a symbol: equity via vnstock, ifc via fmarket, gold, crypto (USD) via CoinGecko.", QueryParams: []RouteParameter{skipCacheParam(), QueryParam("symbol", "string", "Ticker symbol."), QueryParam("type", "string", "Ticker type.")}})
	marketAPI.Get("/exchange-rate", market.GetExchangeRate, RouteMeta{Summary: "Latest exchange rate between two currency codes.", QueryParams: []RouteParameter{skipCacheParam(), QueryParam("from", "string", "Source currency code."), QueryParam("to", "string", "Target currency code.")}})
	marketAPI.Get("/price-series", market.GetPriceSeries, RouteMeta{Summary: "Historical price points for a symbol and series type.", QueryParams: []RouteParameter{skipCacheParam(), QueryParam("symbol", "string", "Ticker symbol."), QueryParam("type", "string", "Series type.")}})
	marketAPI.Get("/candles", market.GetCandles, RouteMeta{Summary: "OHLCV bars for a VN equity or index with the provider that served them and any fallback reason.", QueryParams: []RouteParameter{skipCacheParam(), QueryParam("symbol", "string", "Ticker symbol, e.g. FPT or VNINDEX."), QueryParam("timeframe", "string", "1h, 4h, 1d or 1w.", "1d"), QueryParam("days", "integer", "Lookback in days; defaults to enough history for EMA200.")}})
	marketAPI.Get("/bank-rates", market.GetBankInterestRate, RouteMeta{Summary: "Published bank savings rates.", QueryParams: []RouteParameter{skipCacheParam()}})
}

// RegisterFmarketRoutes exposes the fmarket explorer actions over HTTP
// (WM-MKT-6). Actions only read fmarket data, so they are not write-guarded.
func RegisterFmarketRoutes(api RouteRegistrar, h *FmarketHandler) {
	fmarket := api.Group("/fmarket", "fmarket")
	fmarket.Post("/action", h.Action, RouteMeta{
		Summary:     "Runs an fmarket explorer action: products filter, issuers, product details, NAV history, gold and USD history, gold products.",
		QueryParams: []RouteParameter{forceParam()},
		RequestBody: jsonBody(map[string]any{"action": map[string]any{"type": "string", "enum": domain.FmarketActions()}, "params": map[string]any{"type": "object"}}, "action"),
	})
	fmarket.Get("/health", h.Health, RouteMeta{Summary: "Checks the fmarket provider."})
}
