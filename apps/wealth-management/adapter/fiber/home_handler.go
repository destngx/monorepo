package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"

	"github.com/gofiber/fiber/v2"
)

// Home page (ROADMAP R5, WM-HOME-1). Reads only.

const homeScope = "home_handler"

// NeedsYouSummary documents GET /api/home/needs-you (also the MCP tool).
const NeedsYouSummary = "The home inbox: what needs the user now, ranked. items: id, kind (drafts: quick capture drafts to " +
	"approve; bill-overdue: unpaid bills and card payments past their date, critical; card-due: unpaid card payments due " +
	"within 7 days, serious within 2; budget-over-pace: expense categories over their limit or on pace to pass it by the " +
	"month end, pace from day 7, fixed categories counted as spent; lent-overdue, goal-plan-short, rebalance, idle-cash, emergency-fund-rate: " +
	"the live engine rule findings (as POST /api/recommendations/refresh evaluates them), quiet while the user has the " +
	"rule's recommendation snoozed, dismissed in the last 90 days or done in the last 30; goal-behind: active goals " +
	"behind pace when the plan as a whole is not short; uncategorized: rows of the last 60 days without a known " +
	"category; month-close: in the first 10 days of a month while last month's snapshot is not written, counting the open " +
	"checklist steps of GET /api/close without repeating them; recommendation: the 3 open review recommendations with " +
	"the largest impact, rule items excluded), " +
	"status (critical, serious, warning, info), headline and detail (no names of people or goals), amountVnd (the money " +
	"it is about), impactVnd (estimated VND a year), dueDate, count, route (the app page to act on) and recommendationId. " +
	"Ranked by severity, then impactVnd (else amountVnd) largest first, then the earlier dueDate. unavailable lists " +
	"sources that failed to load (their items are missing); the call fails only when every source fails."

type HomeHandler struct {
	home port.HomeService
	log  *logger.Logger
}

func NewHomeHandler(home port.HomeService, log *logger.Logger) *HomeHandler {
	return &HomeHandler{home: home, log: log}
}

func RegisterHomeRoutes(api RouteRegistrar, h *HomeHandler) {
	home := api.Group("/home", "home")
	home.Get("/needs-you", h.NeedsYou, RouteMeta{Summary: NeedsYouSummary, QueryParams: []RouteParameter{forceParam()}})
}

func (h *HomeHandler) NeedsYou(c *fiber.Ctx) error {
	return respondWith(c, h.log, homeScope)(h.home.NeedsYou(c.UserContext(), forceFresh(c)))
}
