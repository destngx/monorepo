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

func (h *PortfolioHandler) Allocation(c *fiber.Ctx) error {
	return respondWith(c, h.log, "portfolio_handler")(h.portfolio.Allocation(c.UserContext(), forceFresh(c)))
}

// RegisterPortfolioRoutes needs the sheet; without the market providers
// holdings are valued from the sheet only and missing prices are issues.
func RegisterPortfolioRoutes(api RouteRegistrar, h *PortfolioHandler) {
	investments := api.Group("/investments", "investments")
	investments.Get("/valuation", h.Valuation, RouteMeta{
		Summary:     "Values crypto and fund holdings in VND (sheet value, else quantity × sheet or live price) with the USD/USDT rates used, per-holding issues and WM-RULE-8 totals.",
		QueryParams: []RouteParameter{forceParam()},
	})
	investments.Get("/allocation", h.Allocation, RouteMeta{
		Summary:     "The one asset allocation used by Investments, Accounts and Health: lines cash (positive VND account balances), funds (valued fund certificates), crypto (max of crypto accounts and holdings) and moneyLent (owed to the user; no target, out of drift), each with value and share of the total; target classes add current percent, target and drift from the Preferences tab and a status (on_target, over, under, no_target). rebalance is true when a class drifts more than driftThreshold (5) points.",
		QueryParams: []RouteParameter{forceParam()},
	})
}
