package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/port"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Monthly close and net worth history (ROADMAP R8, P3). The snapshot write
// goes through the WriteGuard and touches only the app-managed Snapshots tab.

const monthlyCloseSummary = "Monthly close checklist for a month (default: last month during the first 10 days of a " +
	"month, else this month). items, in order: drafts (capture drafts waiting; count), categorize (the month's " +
	"rows Uncategorized or in a category the Categories tab lacks; count, amount), budget (expense categories over " +
	"their limit; count, amount = over by, target = total limit), goals (amount = the month's income - expense, " +
	"target = what dated goals need a month, count = goals behind; setup needed is an action), moneyLent (people " +
	"owing for 30+ days; count, amount), recommendations (open items of the recommendations list; count, amount = " +
	"their estimated impact a year), snapshot (the month's Snapshots row: done when written for the month end, or " +
	"during the month). state: done, action or unavailable (detail says why); done of checked counts the items. " +
	"canWriteSnapshot is true for this month and last month. detail never carries amounts."

const snapshotSummary = "Writes the month's row to the app-managed Snapshots tab (A Month, B As of, C Net worth, D " +
	"Assets, E Liabilities, F Money lent, G Cash, H Funds, I Crypto, J Income, K Expense, L Savings, M Savings rate, N " +
	"Health score, O-R the savings, emergency, debt and diversification pillars, S Emergency months, T Written, U " +
	"Note), creating the tab on first use. The month's existing row is updated (its note kept), never duplicated: " +
	"201 when a row was added, 200 when updated. Only this month (figures as of today) or last month (balances taken " +
	"back to its last day by undoing later flows; holdings at today's prices). Balances and holdings are read afresh."

const netWorthHistorySummary = "Net worth per month from the first month with data to this month, oldest first. " +
	"source: snapshot (the Snapshots row), estimated (back-calculated from the nearest later known value by the " +
	"flows booked in between; misses price moves) or live (this month: today's net worth). change is against the " +
	"previous month; when both months are measured (not estimated) it is split into savings (income - expense " +
	"booked between their dates, lending left out) and market (the rest: price moves, corrections and anything not " +
	"recorded as income or expense). milestones: the first month each round level (1-2-5 steps from 100 million " +
	"VND) is crossed upwards, and the all-time high; estimated when that month is."

type CloseHandler struct {
	service port.MonthlyCloseService
	log     *logger.Logger
}

func NewCloseHandler(service port.MonthlyCloseService, log *logger.Logger) *CloseHandler {
	return &CloseHandler{service: service, log: log}
}

func (h *CloseHandler) respond(c *fiber.Ctx) func(value any, err error) error {
	return respondWith(c, h.log, "close_handler")
}

func (h *CloseHandler) GetMonthlyClose(c *fiber.Ctx) error {
	return h.respond(c)(h.service.MonthlyClose(c.UserContext(), strings.TrimSpace(c.Query("month")), forceFresh(c)))
}

func (h *CloseHandler) WriteSnapshot(c *fiber.Ctx) error {
	var body struct {
		Month string `json:"month"`
	}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&body); err != nil {
			return badRequest(c, "invalid request body")
		}
	}
	result, err := h.service.WriteSnapshot(c.UserContext(), strings.TrimSpace(body.Month))
	if err != nil {
		return respondError(c, h.log, "close_handler", err)
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	return c.Status(status).JSON(result)
}

func (h *CloseHandler) GetNetWorthHistory(c *fiber.Ctx) error {
	return h.respond(c)(h.service.NetWorthHistory(c.UserContext(), forceFresh(c)))
}

func RegisterMonthlyCloseRoutes(api RouteRegistrar, h *CloseHandler, guard WriteGuard) {
	group := api.Group("/close", "close")
	group.Get("/", h.GetMonthlyClose, RouteMeta{
		Summary:     monthlyCloseSummary,
		QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "yyyy-mm, up to this month.")},
	})
	body := jsonBody(map[string]any{"month": map[string]any{"type": "string", "description": "yyyy-mm: this month or last month; blank = the close month."}})
	body["required"] = false
	group.Post("/snapshot", guard.Wrap(h.WriteSnapshot), RouteMeta{Summary: snapshotSummary, RequestBody: body})
	api.Group("/net-worth", "net-worth").Get("/history", h.GetNetWorthHistory, RouteMeta{
		Summary: netWorthHistorySummary, QueryParams: []RouteParameter{forceParam()},
	})
}
