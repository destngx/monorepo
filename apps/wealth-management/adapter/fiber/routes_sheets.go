package fiber

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"

	"github.com/gofiber/fiber/v2"
)

// Route registration for sheet-backed data and analytics. Every write goes
// through the WriteGuard (WM-SEC-1).

func forceParam() RouteParameter {
	return QueryParam("force", "boolean", "Bypass cache and read the Google Sheet directly.", false)
}

func jsonBody(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return map[string]any{
		"required": true,
		"content":  map[string]any{"application/json": map[string]any{"schema": schema}},
	}
}

var (
	schemaString  = map[string]any{"type": "string"}
	schemaNumber  = map[string]any{"type": "number"}
	schemaBoolean = map[string]any{"type": "boolean"}
	schemaTags    = map[string]any{"type": "array", "items": schemaString}
)

func transactionProperties() map[string]any {
	return map[string]any{
		"accountName":     schemaString,
		"date":            map[string]any{"type": "string", "description": "dd/mm/yyyy or yyyy-mm-dd"},
		"payee":           schemaString,
		"category":        schemaString,
		"tags":            schemaTags,
		"cleared":         schemaBoolean,
		"payment":         schemaNumber,
		"deposit":         schemaNumber,
		"memo":            schemaString,
		"referenceNumber": schemaString,
	}
}

func RegisterSheetRoutes(api RouteRegistrar, h *DatabaseHandler, guard WriteGuard) {
	accounts := api.Group("/accounts", "accounts")
	accounts.Get("/", h.GetAccounts, RouteMeta{Summary: "Lists live accounts (soft-deleted rows excluded) with type, balance and currency. Balances are derived from transactions only.", QueryParams: []RouteParameter{forceParam()}})
	accounts.Post("/", guard.Wrap(h.CreateAccount), RouteMeta{
		Summary:     "Creates an account row. Currency is the entry currency (USDT by default for Binance accounts, else VND), stored in column I; the opening balance and every balance are VND.",
		RequestBody: jsonBody(map[string]any{"name": schemaString, "type": schemaString, "balance": schemaNumber, "currency": schemaString, "note": schemaString}, "name", "type"),
	})
	accounts.Patch("/:name", guard.Wrap(h.UpdateAccount), RouteMeta{
		Summary:     "Edits an account's type and/or note. Balances are derived by the sheet and cannot be edited.",
		PathParams:  []RouteParameter{PathParam("name", "string", "Account name.")},
		RequestBody: jsonBody(map[string]any{"type": schemaString, "note": schemaString}),
	})
	accounts.Delete("/:name", guard.Wrap(h.DeleteAccount), RouteMeta{
		Summary:    "Soft-deletes an account by prefixing its name with [DELETED].",
		PathParams: []RouteParameter{PathParam("name", "string", "Account name.")},
	})

	transactions := api.Group("/transactions", "transactions")
	transactions.Get("/", h.GetTransactions, RouteMeta{Summary: "Lists transactions newest first with categoryType, etag, data issues and effectiveDate (yyyy-mm-dd used for budgets and cash flow: a refund deposit counts in the month named in its memo, any other transaction on its own date; null without a date).", QueryParams: []RouteParameter{forceParam()}})
	transactions.Post("/", guard.Wrap(h.CreateTransaction), RouteMeta{
		Summary:     "Appends a transaction and returns its id. Requires accountName, date, payee, category and payment or deposit. Amounts of USD and USDT accounts (by currency column) are entered in USD and converted to VND by a sheet formula.",
		RequestBody: jsonBody(transactionProperties(), "accountName", "date", "payee", "category"),
	})
	transactions.Post("/batch", guard.Wrap(h.CreateTransactionBatch), RouteMeta{
		Summary: "Appends up to 100 transactions in one write and reports each row. Retrying the same batchId returns the first result without writing again.",
		RequestBody: jsonBody(map[string]any{
			"batchId":      schemaString,
			"transactions": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": transactionProperties()}},
		}, "batchId", "transactions"),
	})
	updateProperties := transactionProperties()
	updateProperties["etag"] = map[string]any{"type": "string", "description": "ETag from the list response; may be sent as If-Match instead."}
	transactions.Patch("/:id", guard.Wrap(h.UpdateTransaction), RouteMeta{
		Summary:     "Replaces a transaction (columns A-J). Returns 409 when the row changed since it was read.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Transaction id, row-N.")},
		RequestBody: jsonBody(updateProperties, "accountName", "date", "payee", "category"),
	})
	transactions.Delete("/:id", guard.Wrap(h.DeleteTransaction), RouteMeta{
		Summary:     "Clears a transaction row. Requires the row's ETag via If-Match or ?etag=; returns 409 when it changed.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Transaction id, row-N.")},
		QueryParams: []RouteParameter{QueryParam("etag", "string", "Row ETag, if not sent as If-Match.")},
	})

	budget := api.Group("/budget", "budget")
	budget.Get("/", h.GetBudget, RouteMeta{Summary: "Reads budget limits from the year's budget tab (BUDGET_SHEET_TEMPLATE, default Budget_{year}).", QueryParams: []RouteParameter{forceParam(), QueryParam("year", "integer", "Budget year; defaults to the current year.")}})
	budget.Patch("/limits", guard.Wrap(h.UpdateBudgetLimit), RouteMeta{
		Summary:     "Sets a category's monthly limit (month = YYYY-MM) or, without month, its yearly limit.",
		RequestBody: jsonBody(map[string]any{"year": map[string]any{"type": "integer"}, "category": schemaString, "month": schemaString, "limit": schemaNumber}, "category", "limit"),
	})
	budget.Post("/reallocate", guard.Wrap(h.ReallocateBudget), RouteMeta{
		Summary:     "Moves amount of a month's limit from one category to another after re-reading the sheet; returns both new limits as {from:{category,limit},to:{category,limit}}. 400 when amount <= 0, the categories are the same or from has less limit than amount.",
		RequestBody: jsonBody(map[string]any{"year": map[string]any{"type": "integer"}, "month": map[string]any{"type": "string", "description": "YYYY-MM"}, "from": schemaString, "to": schemaString, "amount": schemaNumber}, "month", "from", "to", "amount"),
	})

	categories := api.Group("/categories", "categories")
	categories.Get("/", h.GetCategories, RouteMeta{Summary: "Lists income, expense and non-budget categories with their nature (fixed, essential, flexible, lending; omitted when untagged), definition and split tags (omitted when empty).", QueryParams: []RouteParameter{forceParam()}})
	categories.Patch("/nature", guard.Wrap(h.UpdateCategoryNature), RouteMeta{
		Summary: "Sets a category's nature in Categories column B and returns the category. The category is matched by exact name; 404 when it is missing, 409 when it appears more than once, 400 for an unknown nature. Lending categories hold money lent and repaid: they leave income and expense.",
		RequestBody: jsonBody(map[string]any{
			"category": schemaString,
			"nature":   map[string]any{"type": "string", "enum": []string{"fixed", "essential", "flexible", "lending", "none"}, "description": "none clears the nature."},
		}, "category", "nature"),
	})
	categories.Patch("/definition", guard.Wrap(h.UpdateCategoryDefinition), RouteMeta{
		Summary: "Sets a category's definition (Categories column D: what belongs in it, used when the AI files transactions) and split tags (column E: tags that divide its spend) and returns the category. Empty values clear them. The category is matched by exact name; 404 when it is missing, 409 when it appears more than once, 400 for a definition over 500 characters, a split tag over 30 characters or with a comma, or more than 5 split tags.",
		RequestBody: jsonBody(map[string]any{
			"category":   schemaString,
			"definition": schemaString,
			"splitTags":  map[string]any{"type": "array", "items": schemaString, "description": "Stored trimmed and lower case."},
		}, "category"),
	})
	registerGoalWriteRoutes(api.Group("/goals", "goals"), h, guard)
	registerBillsWriteRoutes(api, h, guard)
	api.Group("/tags", "tags").Get("/", h.GetTags, RouteMeta{Summary: "Lists normalised transaction tags.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/investments", "investments").Get("/assets", h.GetInvestmentAssets, RouteMeta{Summary: "Returns crypto and fund holdings, raw and typed (symbol, quantity, price, value).", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/cards", "cards").Get("/programs", h.GetCardPrograms, RouteMeta{Summary: "Card configuration from the sheet: programmes with their dated versions and cashback rules (Programs, ProgramRules), banks with the shared account and reference limit (Banks), cards held (Cards) and payee MCC mappings (Merchants). A missing tab contributes no rows; 400 naming each invalid row.", QueryParams: []RouteParameter{forceParam()}})
	capture := api.Group("/capture", "capture")
	capture.Get("/rules", h.GetCaptureRules, RouteMeta{Summary: "Lists Smart Capture account aliases and payee categories from the CaptureRules tab (empty until the first save).", QueryParams: []RouteParameter{forceParam()}})
	capture.Post("/rules", guard.Wrap(h.SaveCaptureRules), RouteMeta{
		Summary: "Upserts Smart Capture rules by kind and pattern (case and diacritics ignored), creating the CaptureRules tab on first use.",
		RequestBody: jsonBody(map[string]any{"rules": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
			"kind":    map[string]any{"type": "string", "enum": []string{"alias", "payee"}},
			"pattern": map[string]any{"type": "string", "description": "Account hint (alias) or payee (payee)."},
			"value":   map[string]any{"type": "string", "description": "Account name (alias) or category (payee)."},
		}}}}, "rules"),
	})
	preferences := api.Group("/preferences", "preferences")
	preferences.Get("/", h.GetPreferences, RouteMeta{
		Summary:     "Reads the app-managed Preferences tab (A Key, B Value, C Note): allocationTargets (cash, funds, crypto percent; null until all three are set and add up to 100), goalPool (emergencyMonths default 6, spendingBufferMonths default 1), safeToSpend (notReadyAccounts: accounts left out of safe-to-spend's ready money, from safeToSpend.notReadyAccounts, a JSON list of account names; etag for POST /api/bills/safe-to-spend/accounts), accountMinimums (accounts: account, minimum in VND, note, from accounts.minimumBalance, a JSON object of account name to {minimum, note}; etag for POST /api/bills/safe-to-spend/minimum-balance), the rows of known keys and issues naming ignored rows. tabExists is false before the first save.",
		QueryParams: []RouteParameter{forceParam()},
	})
	preferences.Post("/", guard.Wrap(h.SavePreferences), RouteMeta{
		Summary: "Saves preference keys: updates the key's row or appends it, creating the Preferences tab on the first save; writes no other tab. A null value clears the key. The three allocation targets are saved together and must add up to 100. safeToSpend.notReadyAccounts is set per account with POST /api/bills/safe-to-spend/accounts, accounts.minimumBalance with POST /api/bills/safe-to-spend/minimum-balance instead. Returns the preferences read back.",
		RequestBody: jsonBody(map[string]any{"values": map[string]any{"type": "array", "minItems": 1, "items": map[string]any{"type": "object", "required": []string{"key", "value"}, "properties": map[string]any{
			"key":   map[string]any{"type": "string", "enum": domain.PreferenceKeyNames()},
			"value": map[string]any{"type": "number", "nullable": true, "description": "Percent for allocation targets (0-100), months for goal keys (emergencyMonths 1-24, spendingBufferMonths 0-12); null clears."},
		}}}}, "values"),
	})
	api.Group("/sync", "sync").Post("/", guard.Wrap(h.Sync), RouteMeta{Summary: "Invalidates every sheet-derived cache key so the next reads come from the sheet, and records lastSyncAt."})
	api.Group("/sheets", "sheets").Get("/info", h.GetSheetInfo, RouteMeta{Summary: "Returns {spreadsheetId, title, lastSyncAt}; lastSyncAt is the RFC3339 time of the last successful sync, null before the first."})
	api.Group("/init", "init").Get("/", h.Init, RouteMeta{Summary: "Warms AI prompts and knowledge from the sheet and returns row counts.", QueryParams: []RouteParameter{forceParam()}})
}

func RegisterAnalyticsRoutes(api RouteRegistrar, h *AnalyticsHandler) {
	analytics := api.Group("/analytics", "analytics")
	analytics.Get("/dashboard", h.GetDashboard, RouteMeta{Summary: "Dashboard summary: balance sheet, net worth change this month, month-to-date income and expense against the same days of last month, month-end expense pace, 30-day net worth trend, cash flow history and top accounts.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/budget", h.GetBudgetReport, RouteMeta{Summary: "Budget actuals for a month: spent, remaining and status per category, with burn rate and, for the current month, burnOutDate (yyyy-mm-dd the limit runs out at the current pace; null when it will not this month).", QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "YYYY-MM; defaults to the current month.")}})
	analytics.Get("/budget/spending", h.GetSpendingWindows, RouteMeta{Summary: "How spending is going (WM-BUD-12): today against a typical day (day-to-day average of the 30 days before), the last 30 days and the last 3 months (90 days), each ending today, with fixed costs apart from day-to-day spend, daily and monthly averages, the previous period of the same length, expense categories with their split tags, the last 90 days day by day, and recommendations (faster, slower, category_rising, split_rising, heavy_day).", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/budget/history", h.GetBudgetHistory, RouteMeta{Summary: "Limit and spending per expense category for the months ending at month, oldest first: {months:[YYYY-MM], rows:[{category, values:[{month, limit, spent}]}]}.", QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "Last month, YYYY-MM; defaults to the current month."), QueryParam("months", "integer", "Number of months, 1-24.", defaultBudgetHistory)}})
	analytics.Get("/cash-flow/yearly", h.GetYearlyCashFlow, RouteMeta{Summary: "Income and expense of a calendar year: per month (January first) with cash flow and savings rate, year total, monthly average over the elapsed months, the same months of the previous year with % change, per category per month (largest first), and the years that have data (newest first).", QueryParams: []RouteParameter{forceParam(), QueryParam("year", "integer", "Calendar year; defaults to the current year.")}})
	analytics.Get("/cards", h.GetCards, RouteMeta{Summary: "Credit cards per bank: the shared limit, usage (what the billed accounts owe) and remaining room; per card the current period's standing (spend, confirmed and estimated cashback, caps left, next tier), the pay-by date and 36 periods of history reconciled with the refunds received.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/cards/suggest", h.SuggestCards, RouteMeta{Summary: "Ranks the held cards for a payment by the cashback it would add to each card's current period, cards whose bank has room for it first. The MCC comes from the mcc parameter (confirmed), else the payee's Merchants mapping, else it is unknown and only fallback rules apply.", QueryParams: []RouteParameter{forceParam(), QueryParam("payee", "string", "Payee of the payment; required without mcc."), QueryParam("amount", "number", "Payment amount in VND, positive."), QueryParam("mcc", "integer", "Merchant category code, when known.")}})
	analytics.Get("/cards/compare", h.ComparePrograms, RouteMeta{Summary: "Replays the held cards' payments of the last 12 months under every programme's current terms, held or not, ranked by cashback less the annual fee.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/health", h.GetFinancialHealth, RouteMeta{Summary: "Financial health score with per-pillar formulas and recommendations. Debt-to-income is the card minimum payments (5% of each negative account balance) / average monthly income (reference 35%). projection gives net worth in 5, 10 and 20 years, nominal and inflation-adjusted (real).", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/account-histories", h.GetAccountHistories, RouteMeta{Summary: "Back-calculated daily balances per account.", QueryParams: []RouteParameter{forceParam(), QueryParam("days", "integer", "Number of days, 1-366.", defaultHistoryDays)}})
	api.Group("/goals", "goals").Get("/", h.GetGoalPlan, RouteMeta{Summary: goalPlanSummary, QueryParams: []RouteParameter{forceParam()}})
	registerBillsReadRoutes(api, h)
	api.Group("/lending", "lending").Get("/", h.GetMoneyLent, RouteMeta{Summary: "Money lent per person from transactions in lending categories (payee = person; payment = lent, deposit = repaid): lent, repaid, outstanding (negative = the user owes that person), lastActivity, openSince and daysOutstanding of the open balance, and the person's transactions newest first. Totals owedToYou, youOwe, peopleOwing, peopleYouOwe, oldestDaysOutstanding. People owing the user come first, settled people (outstanding 0) last.", QueryParams: []RouteParameter{forceParam()}})
}

// RegisterUnconfiguredSheetRoutes registers the sheet and analytics routes
// when the Google Sheet cannot be used, each answering with cause (503
// SHEETS_NOT_CONFIGURED naming the missing variables when config is absent).
func RegisterUnconfiguredSheetRoutes(api RouteRegistrar, cause error, log *logger.Logger) {
	unavailable := api.Answering(func(c *fiber.Ctx) error {
		return respondError(c, log, "sheets", cause)
	})
	RegisterSheetRoutes(unavailable, NewDatabaseHandler(nil, log), WriteGuard{})
	RegisterAnalyticsRoutes(unavailable, NewAnalyticsHandler(nil, log))
	RegisterPortfolioRoutes(unavailable, NewPortfolioHandler(nil, log))
	RegisterRecommendationRoutes(unavailable, NewRecommendationHandler(nil, log), WriteGuard{})
	RegisterMonthlyCloseRoutes(unavailable, NewCloseHandler(nil, log), WriteGuard{})
	RegisterHomeRoutes(unavailable, NewHomeHandler(nil, log))
}
