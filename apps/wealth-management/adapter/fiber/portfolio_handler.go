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

func (h *PortfolioHandler) Exposure(c *fiber.Ctx) error {
	return respondWith(c, h.log, "portfolio_handler")(h.portfolio.Exposure(c.UserContext(), forceFresh(c)))
}

func (h *PortfolioHandler) Hedges(c *fiber.Ctx) error {
	return respondWith(c, h.log, "portfolio_handler")(h.portfolio.Hedges(c.UserContext(), forceFresh(c)))
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
	investments.Get("/exposure", h.Exposure, RouteMeta{
		Summary:     "My exposure (ROADMAP M1/M2) from the one allocation: classes (cash, funds, crypto, moneyLent) with value and share; volatile = funds + crypto, volatileShare their percent of the total. Today's change applies each holding's latest move to its VND value: coins over about 24h (hourly bars of <COIN>-USD), funds since the previous published NAV; covered is the value the change is based on. crypto: held coins summed per symbol with price, move and change, share of net worth (balance sheet) and the fixed drawdown scenarios (30% and 50%): loss, net worth after and its change. Market failures are issues.",
		QueryParams: []RouteParameter{forceParam()},
	})
	investments.Get("/hedges", h.Hedges, RouteMeta{
		Summary:     "Gold and USD as protection (ROADMAP M4): the cash class value and share (cashShare) and the VND share (everything but crypto), with a year of daily SJC gold (sell price, VND per tael) and USD/VND closes from the market data server: latest close, the change over 3M and 1Y (partial when the history is shorter) and onCash, what the VND cash would have gained or lost held in the asset instead.",
		QueryParams: []RouteParameter{forceParam()},
	})
}
