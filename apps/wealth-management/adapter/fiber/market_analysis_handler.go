package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type MarketAnalysisHandler struct {
	analysis port.MarketAnalysisService
	log      *logger.Logger
}

func NewMarketAnalysisHandler(analysis port.MarketAnalysisService, log *logger.Logger) *MarketAnalysisHandler {
	return &MarketAnalysisHandler{analysis: analysis, log: log}
}

func (h *MarketAnalysisHandler) Pulse(c *fiber.Ctx) error {
	timeframe, err := domain.ParseTimeframe(c.Query("timeframe", string(domain.Timeframe1D)))
	if err != nil {
		return badRequest(c, err.Error())
	}
	ctx := domain.WithBypassCache(c.UserContext(), forceFresh(c))
	pulse, err := h.analysis.Pulse(ctx, timeframe)
	if err != nil {
		return upstreamError(c, h.log, "market_analysis_handler: pulse", err)
	}
	return c.JSON(pulse)
}

func (h *MarketAnalysisHandler) Analysis(c *fiber.Ctx) error {
	symbol := strings.TrimSpace(c.Query("symbol"))
	if symbol == "" {
		return badRequest(c, "symbol is required")
	}
	timeframe, err := domain.ParseTimeframe(c.Query("timeframe", string(domain.Timeframe1D)))
	if err != nil {
		return badRequest(c, err.Error())
	}
	ctx := domain.WithBypassCache(c.UserContext(), forceFresh(c))
	analysis, err := h.analysis.Analysis(ctx, symbol, timeframe)
	if err != nil {
		return upstreamError(c, h.log, "market_analysis_handler: analysis", err)
	}
	return c.JSON(analysis)
}

// RegisterMarketAnalysisRoutes adds the deterministic market views. They
// read candles only, so they work without the sheet.
func RegisterMarketAnalysisRoutes(api RouteRegistrar, h *MarketAnalysisHandler) {
	market := api.Group("/market", "market")
	timeframe := QueryParam("timeframe", "string", "1h, 4h, 1d or 1w.", "1d")
	force := QueryParam("force", "boolean", "Bypass the market cache and fetch fresh candles.", false)
	market.Get("/pulse", h.Pulse, RouteMeta{
		Summary:     "VN market pulse: per-instrument change, direction and momentum, top movers, return correlations, capital flow, scenario, VN-Index technicals and sentiment (WM-RULE-9). Failed instruments carry their error.",
		QueryParams: []RouteParameter{timeframe, force},
	})
	market.Get("/analysis", h.Analysis, RouteMeta{
		Summary:     "Ticker deep-dive: technicals for the timeframe, 1h/4h/1d/1w confluence, seasonality and history-only risk statistics with a bootstrap simulation. Signals are rule-based heuristics.",
		QueryParams: []RouteParameter{QueryParam("symbol", "string", "Ticker or instrument, e.g. FPT, VNINDEX, SJC, USDVND."), timeframe, force},
	})
}
