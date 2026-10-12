package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"

	"github.com/gofiber/fiber/v2"
)

type PortfolioHandler struct {
	portfolio port.PortfolioService
	log       *logger.Logger
}

func NewPortfolioHandler(portfolio port.PortfolioService, log *logger.Logger) *PortfolioHandler {
	return &PortfolioHandler{portfolio: portfolio, log: log}
}

func (h *PortfolioHandler) Valuation(c *fiber.Ctx) error {
	return respondWith(c, h.log, "portfolio_handler")(h.portfolio.Valuation(c.UserContext(), forceFresh(c)))
}

// RegisterPortfolioRoutes needs both the sheet and the market providers.
func RegisterPortfolioRoutes(api RouteRegistrar, h *PortfolioHandler) {
	api.Group("/investments", "investments").Get("/valuation", h.Valuation, RouteMeta{
		Summary:     "Values crypto and fund holdings in VND (sheet value, else quantity × sheet or live price) with the USD/USDT rates used, per-holding issues and WM-RULE-8 totals.",
		QueryParams: []RouteParameter{forceParam()},
	})
}
