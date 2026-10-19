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
	"safeToSpend (D33, D36), in tiers: committed = billsDue + cardPayments + fixedCosts (never spendable); amount = " +
	"readyMoney - committed (negative: shortBy); essentials kept for needs; safeToSpend = free to spend = amount - " +
	"essentials, floored at 0 (tightBy: the essentials left uncovered). fixed: per fixed category (Categories nature) " +
	"and month the window touches, expected (the monthly budget limit, basis budget, else the 6-completed-month average, " +
	"basis average) - spent that month - inBills (its unpaid recurring bills that month, matched by the item's category " +
	"or its payments' category), floored at 0 as amount. essential: per essential category, perDay (6-completed-month " +
	"average daily spending without recurring bills, basis average, else the monthly budget / days in month, basis " +
	"budget) x days. natureTagged false: no category is tagged fixed or essential and nothing is set aside; " +
	"setAsideRule states the rule. readyMoney = positive balances of active use, rarely use, " +
	"bank and cash accounts (readyAccounts), without crypto accounts or accounts marked not ready to use " +
	"(excludedAccounts with reason crypto or not_ready; readiness holds the saved names and the etag to change them). " +
	"An account with a minimum balance to keep (D35, accounts.minimumBalance) adds only max(0, balance - minimum): " +
	"each ready account has balance, reserved and spendable, reserved is the total kept back (a hard reserve, since a " +
	"bank reward on the average monthly balance cannot be made up late in the month). minimumBalance holds the saved " +
	"minimums and the etag to change them; minimumBalances track every account with a minimum this month (minimumRule): " +
	"sumSoFar and averageSoFar over the finished days (1st to yesterday, rebuilt backwards from today's balance with the " +
	"account's transactions), projectedAverage = (sumSoFar + balance x daysLeft) / daysInMonth, requiredBalance = " +
	"(minimum x daysInMonth - sumSoFar) / daysLeft (today included), extraNeeded and status on_track or at_risk; " +
	"unknownMinimum names minimums for no account (ignored). " +
	"billsDue and cardPayments: unpaid, from the 1st of this month (overdue ones too) to the day before until, the " +
	"next unpaid income (a payment due on payday is paid by that wage); without an income, the next 30 days up to " +
	"and including until. No goal money is subtracted. perDay = safeToSpend / days; safeToSpend.issues names marked accounts that no longer exist (ignored). " +
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
		Summary:     "Headline for the home page: safeToSpend (every component and tier, as in GET /api/bills), facts (active items, unpaid bills and card payments in the next 30 days, overdue count and amount, committed, fixed costs, essentials, free to spend with shortBy and tightBy, suggestions waiting), overdue bills and unpaid bills due within 7 days.",
		QueryParams: []RouteParameter{forceParam()},
	})
	bills.Get("/idle-cash", h.GetIdleCash, RouteMeta{Summary: idleCashSummary, QueryParams: []RouteParameter{forceParam()}})
}

const accountReadinessSummary = "Marks an account ready to use or not for safe-to-spend (D33): ready false adds it to " +
	"safeToSpend.notReadyAccounts in the app-managed Preferences tab (a JSON list of account names; the tab is " +
	"created on first use and no other tab is written), ready true removes it (names compared ignoring case; a name " +
	"that is no longer an account can be removed). Returns the preferences read back. 400 without account or etag, " +
	"404 when marking an unknown account not ready, 409 when the setting changed since its etag was read."

const emergencyAccountSummary = "Marks an account as an emergency fund account or not for the idle cash check (D34): " +
	"emergency true adds it to idleCash.emergencyAccounts in the app-managed Preferences tab (a JSON list of account " +
	"names; the tab is created on first use and no other tab is written), false removes it (names compared ignoring " +
	"case; a name that is no longer an account can be removed). Emergency accounts cover the emergency fund target " +
	"first and are never called idle. Returns the preferences read back. 400 without account or etag, 404 when " +
	"marking an unknown account, 409 when the setting changed since its etag was read."

const accountMinimumSummary = "Sets or removes an account's minimum balance to keep (D35), e.g. the average monthly " +
	"balance a bank asks for a reward: minimum (VND) and an optional note are saved in accounts.minimumBalance in the " +
	"app-managed Preferences tab (a JSON object of account name to {minimum, note}; the tab is created on first use and " +
	"no other tab is written). A blank (null) or 0 minimum removes the entry, also for a name that is no longer an " +
	"account. The minimum is never ready to spend nor idle, and the month-to-date average is tracked against it. " +
	"Returns the preferences read back. 400 without account or etag or for a negative minimum or a note over 200 " +
	"characters, 404 when setting a minimum on an unknown account, 409 when the setting changed since its etag was read."

const idleCashSummary = "Idle cash check (ROADMAP R6, D34), for the bills and home pages: idle = max(0, " +
	"components.readyMoney - keepLiquid - dayToDayFloat). readyMoney: safe-to-spend's ready money (active, rarely used, " +
	"bank and cash accounts; no crypto, no accounts marked not ready; excluded lists them with the reason) without the " +
	"emergency accounts. keepLiquid = billsDue + cardPayments (unpaid in the next 30 days, overdue too) + goalSetAside " +
	"(the goal plan's full required per month) + emergencyFromReady (the emergency fund target, the emergency goal's " +
	"target else emergency months x monthly needs, minus emergencyCovered = what the emergency accounts hold, up to the " +
	"target). dayToDayFloat: the idleCash.dayToDayFloat preference (floatFromPreference true), else the spending buffer " +
	"(bufferMonths x averageMonthlyExpense). accounts: the everyday ready accounts lowest rate first, rate read from the " +
	"account note (\"earn rate 4%\", \"lãi 4%\", \"rate 4.5%\"; else 0) and the idle part assumed in each. currentRate: " +
	"weighted over those parts. best: the best instant-access rate only, a no-term deposit (access no_term, withdraw " +
	"any time) or a 1-month deposit (access one_month, rolls monthly; early withdrawal pays the no-term rate), with " +
	"accessLabel; longer terms and rates without a tenor are never used (null with ratesIssue). ladder (1-month best " +
	"only): parts (2-3, each at least 10.000.000 VND), partAmount, spacingDays. extraPerYear = idle x (best - current) " +
	"/ 100, 0 when not positive. verdict: worth_moving (gap >= 0.5 points and extra >= 200.000 VND a year), " +
	"not_worth_it, nothing_idle or no_rate, with reasons. emergency: the emergency accounts (never idle), balance, " +
	"target, currentRate, extraPerYear and betterRate (the best instant-access rate beats them by the same thresholds), " +
	"settings (names and the etag to change them) and unknown names; issues names marked accounts that are ignored. " +
	"Advice is about rate only: goal money stays pooled."

func registerBillsWriteRoutes(api RouteRegistrar, h *DatabaseHandler, guard WriteGuard) {
	bills := api.Group("/bills", "bills")
	bills.Post("/idle-cash/emergency-accounts", guard.Wrap(h.SetEmergencyAccount), RouteMeta{
		Summary: emergencyAccountSummary,
		RequestBody: jsonBody(map[string]any{
			"account":   map[string]any{"type": "string", "description": "Account name as in the Accounts tab."},
			"emergency": map[string]any{"type": "boolean", "description": "true = holds the emergency fund (covers its target first, never idle)."},
			"etag":      map[string]any{"type": "string", "description": "emergency.settings.etag from GET /api/bills/idle-cash (or idleCash.etag from GET /api/preferences); may be sent as If-Match instead."},
		}, "account", "emergency"),
	})
	bills.Post("/safe-to-spend/minimum-balance", guard.Wrap(h.SetAccountMinimumBalance), RouteMeta{
		Summary: accountMinimumSummary,
		RequestBody: jsonBody(map[string]any{
			"account": map[string]any{"type": "string", "description": "Account name as in the Accounts tab."},
			"minimum": map[string]any{"type": "number", "nullable": true, "minimum": 0, "description": "VND to keep; null or 0 removes the minimum."},
			"note":    map[string]any{"type": "string", "description": "Why it is kept, e.g. monthly points reward (at most 200 characters)."},
			"etag":    map[string]any{"type": "string", "description": "safeToSpend.minimumBalance.etag from GET /api/bills (or accountMinimums.etag from GET /api/preferences); may be sent as If-Match instead."},
		}, "account"),
	})
	bills.Post("/safe-to-spend/accounts", guard.Wrap(h.SetAccountReadiness), RouteMeta{
		Summary: accountReadinessSummary,
		RequestBody: jsonBody(map[string]any{
			"account": map[string]any{"type": "string", "description": "Account name as in the Accounts tab."},
			"ready":   map[string]any{"type": "boolean", "description": "false = not ready to use (slow to withdraw), left out of ready money."},
			"etag":    map[string]any{"type": "string", "description": "safeToSpend.readiness.etag from GET /api/bills (or safeToSpend.etag from GET /api/preferences); may be sent as If-Match instead."},
		}, "account", "ready"),
	})
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

func (h *AnalyticsHandler) GetIdleCash(c *fiber.Ctx) error {
	return h.respond(c)(h.analytics.IdleCash(c.UserContext(), forceFresh(c)))
}

func (h *DatabaseHandler) SetAccountReadiness(c *fiber.Ctx) error {
	var body domain.AccountReadinessInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return h.respond(c)(h.databaseService.SetAccountReadiness(body))
}

func (h *DatabaseHandler) SetAccountMinimumBalance(c *fiber.Ctx) error {
	var body domain.AccountMinimumInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return h.respond(c)(h.databaseService.SetAccountMinimumBalance(body))
}

func (h *DatabaseHandler) SetEmergencyAccount(c *fiber.Ctx) error {
	var body domain.EmergencyAccountInput
	if err := c.BodyParser(&body); err != nil {
		return badRequest(c, "invalid request body")
	}
	body.ETag = etagFrom(c, body.ETag)
	return h.respond(c)(h.databaseService.SetEmergencyAccount(body))
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
