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

func (h *PortfolioHandler) FundPerformance(c *fiber.Ctx) error {
	return respondWith(c, h.log, "portfolio_handler")(h.portfolio.FundPerformance(c.UserContext(), forceFresh(c)))
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
	investments.Get("/fund-performance", h.FundPerformance, RouteMeta{
		Summary:     "Fund holdings vs a bank deposit (WM-INV-21/22): each holding valued at Total Unit × the latest fmarket NAV; when its History (d-m-yyyy,units entries) adds up to Total Unit, every trade is priced at the NAV it was matched at (that day, else the next published) to give cash flows, cost basis (average cost), gain, simple return and the money-weighted annualised return (XIRR, left out under 30 days). costStatus says why a cost is unknown (no_history, history_invalid, units_mismatch, nav_unavailable). The deposit comparison grows the same cash flows at today's best published 12-month (or nearest tenor) bank rate, a labelled proxy since historical deposit rates are not published: depositValue, differenceVND, differencePoints (annualised return minus the rate) and vsDeposit (ahead, behind, even within 0.5% of the money bought). funds aggregates per fund code; totals cover holdings with a known cost.",
		QueryParams: []RouteParameter{forceParam()},
	})
}
