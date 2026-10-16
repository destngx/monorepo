package fiber

import (
	"apps/wealth-management/domain"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

// Bills and safe-to-spend (ROADMAP R4). Reads are analytics; writes go
// through the WriteGuard and touch only the app-managed Recurring tab.

const billsSummary = "Bills and safe-to-spend. items: the Recurring tab rows (kind income, bill, subscription, insurance, " +
	"card; frequency monthly, quarterly, yearly; day; month for quarterly/yearly; match = payee text that marks a " +
	"transaction as the item, whole words, case and accents ignored; category and account must agree when set), each " +
	"with current (the occurrence whose cycle holds today: the calendar month for monthly items, the month before to " +
	"the month after for the others) settled by matching transactions, nextDate, lastMatch and matchCount. " +
	"suggestions: groups of payee + category + direction seen in 3+ of the last 6 completed months with a stable " +
	"amount (within 15% of the median in 2/3 of them) and a typical day (within 5 days), or in the same month of 2+ " +
	"consecutive years (yearly); confirm one by key. calendar: occurrences from the 1st of this month to 30 days " +
	"ahead with status paid, due or overdue, plus one card payment per credit card account due on the pay-by date " +
	"(29th, 28th in February) for its statement (WM-CARD-15), or the balance owed when the card has no settings. " +
	"safeToSpend: spendableCash (active, rarely used, bank and cash accounts) - billsDue (unpaid bills from the 1st of " +
	"this month to until) - cardPayments (due by until) - goalSetAside (the goal plan's full monthly requirement); " +
	"until = the next unpaid income, else 30 days. safeToSpend is floored at 0 and shortBy shows the shortfall. " +
	"issues lists rejected rows and card rows without a card account. tabExists is false until the first item."

func recurringProperties() map[string]any {
	return map[string]any{
		"name":      schemaString,
		"kind":      map[string]any{"type": "string", "enum": []string{"income", "bill", "subscription", "insurance", "card"}},
		"amount":    map[string]any{"type": "number", "description": "VND, positive. Omit on card rows to use the statement balance."},
		"frequency": map[string]any{"type": "string", "enum": []string{"monthly", "quarterly", "yearly"}, "description": "Default monthly."},
		"day":       map[string]any{"type": "integer", "description": "Day of month 1-31 (clamped to short months); not needed on card rows."},
		"month":     map[string]any{"type": "integer", "description": "1-12: first month of a quarterly item, month of a yearly one."},
		"account":   schemaString,
		"category":  schemaString,
		"match":     map[string]any{"type": "string", "description": "Payee text that marks a transaction as this item. Without it, the category and an amount within 15% must match."},
		"status":    map[string]any{"type": "string", "enum": []string{"active", "paused"}, "description": "Default active."},
		"note":      schemaString,
	}
}

func registerBillsReadRoutes(api RouteRegistrar, h *AnalyticsHandler) {
	bills := api.Group("/bills", "bills")
	bills.Get("/", h.GetBills, RouteMeta{Summary: billsSummary, QueryParams: []RouteParameter{forceParam()}})
	bills.Get("/safe-to-spend", h.GetSafeToSpend, RouteMeta{
		Summary:     "Headline for the home page: safeToSpend (every component, as in GET /api/bills), facts (active items, unpaid bills and card payments in the next 30 days, overdue count and amount, suggestions waiting), overdue bills and unpaid bills due within 7 days.",
		QueryParams: []RouteParameter{forceParam()},
	})
}

func registerBillsWriteRoutes(api RouteRegistrar, h *DatabaseHandler, guard WriteGuard) {
	bills := api.Group("/bills", "bills")
	bills.Post("/items", guard.Wrap(h.CreateRecurring), RouteMeta{
		Summary:     "Adds a recurring item to the Recurring tab, creating the tab with its header on first use, and returns it with its id and etag.",
		RequestBody: jsonBody(recurringProperties(), "name", "kind"),
	})
	confirmProperties := recurringProperties()
	delete(confirmProperties, "frequency")
	delete(confirmProperties, "month")
	confirmProperties["key"] = map[string]any{"type": "string", "description": "Suggestion key from GET /api/bills."}
	confirmProperties["status"] = map[string]any{"type": "string", "enum": []string{"active", "paused"}, "description": "paused records a dismissed suggestion so it is not suggested again."}
	bills.Post("/suggestions/confirm", guard.Wrap(h.ConfirmRecurringSuggestion), RouteMeta{
		Summary:     "Writes a current suggestion to the Recurring tab (created on first use); fields given override the suggested ones. 404 when the key is no longer suggested.",
		RequestBody: jsonBody(confirmProperties, "key"),
	})
	updateProperties := recurringProperties()
	updateProperties["etag"] = map[string]any{"type": "string", "description": "ETag from GET /api/bills; may be sent as If-Match instead."}
	bills.Patch("/items/:id", guard.Wrap(h.UpdateRecurring), RouteMeta{
		Summary:     "Replaces a recurring item (columns A-K). Returns 409 when the row changed since it was read.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Item id, recurring-N.")},
		RequestBody: jsonBody(updateProperties, "name", "kind"),
	})
	bills.Post("/items/:id/status", guard.Wrap(h.SetRecurringStatus), RouteMeta{
		Summary:    "Pauses or resumes a recurring item (writes only its Status column). Returns 409 when the row changed since it was read.",
		PathParams: []RouteParameter{PathParam("id", "string", "Item id, recurring-N.")},
		RequestBody: jsonBody(map[string]any{
			"status": map[string]any{"type": "string", "enum": []string{"active", "paused"}},
			"etag":   map[string]any{"type": "string", "description": "ETag from GET /api/bills; may be sent as If-Match instead."},
		}, "status"),
	})
	bills.Delete("/items/:id", guard.Wrap(h.DeleteRecurring), RouteMeta{
		Summary:     "Clears a recurring item row. Requires the row's ETag via If-Match or ?etag=; returns 409 when it changed.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Item id, recurring-N.")},
		QueryParams: []RouteParameter{QueryParam("etag", "string", "Row ETag, if not sent as If-Match.")},
	})
}

func (h *AnalyticsHandler) GetBills(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.Bills(forceFresh(c)))
}

func (h *AnalyticsHandler) GetSafeToSpend(c *fiber.Ctx) error {
	bills, err := h.analytics.Bills(forceFresh(c))
	if err != nil {
		return h.respond(c)(nil, err)
	}
	return c.JSON(bills.Summary())
}

func (h *DatabaseHandler) CreateRecurring(c *fiber.Ctx) error {
	var body domain.RecurringInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	item, err := h.databaseService.CreateRecurring(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(item)
}

func (h *DatabaseHandler) ConfirmRecurringSuggestion(c *fiber.Ctx) error {
	var body domain.RecurringConfirmInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	item, err := h.databaseService.ConfirmRecurringSuggestion(body)
	if err != nil {
		return h.fail(c, err)
	}
	return c.Status(http.StatusCreated).JSON(item)
}

func (h *DatabaseHandler) UpdateRecurring(c *fiber.Ctx) error {
	var body domain.RecurringUpdateInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return h.respond(c)(h.databaseService.UpdateRecurring(pathParam(c, "id"), body))
}

func (h *DatabaseHandler) SetRecurringStatus(c *fiber.Ctx) error {
	var body domain.RecurringStatusInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return h.respond(c)(h.databaseService.SetRecurringStatus(pathParam(c, "id"), body))
}

func (h *DatabaseHandler) DeleteRecurring(c *fiber.Ctx) error {
	etag := c.Query("etag")
	if header := c.Get(fiber.HeaderIfMatch); header != "" {
		etag = header
	}
	if err := h.databaseService.DeleteRecurring(pathParam(c, "id"), strings.Trim(utils.CopyString(etag), `"`)); err != nil {
		return h.fail(c, err)
	}
	return c.JSON(fiber.Map{"success": true})
}

// etagFrom prefers the body's etag, else the If-Match header, unquoted.
func etagFrom(c *fiber.Ctx, body string) string {
	if body == "" {
		body = utils.CopyString(c.Get(fiber.HeaderIfMatch))
	}
	return strings.Trim(body, `"`)
}
