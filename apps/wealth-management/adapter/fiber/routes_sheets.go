package fiber

import (
	"apps/wealth-management/adapter/logger"

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

	api.Group("/categories", "categories").Get("/", h.GetCategories, RouteMeta{Summary: "Lists income, expense and non-budget categories.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/goals", "goals").Get("/", h.GetGoals, RouteMeta{Summary: "Lists savings goals with progress and status.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/loans", "loans").Get("/", h.GetLoans, RouteMeta{Summary: "Lists loans and debts with monthly and yearly progress.", QueryParams: []RouteParameter{forceParam()}})
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
	api.Group("/sync", "sync").Post("/", guard.Wrap(h.Sync), RouteMeta{Summary: "Invalidates every sheet-derived cache key so the next reads come from the sheet, and records lastSyncAt."})
	api.Group("/sheets", "sheets").Get("/info", h.GetSheetInfo, RouteMeta{Summary: "Returns {spreadsheetId, title, lastSyncAt}; lastSyncAt is the RFC3339 time of the last successful sync, null before the first."})
	api.Group("/init", "init").Get("/", h.Init, RouteMeta{Summary: "Warms AI prompts and knowledge from the sheet and returns row counts.", QueryParams: []RouteParameter{forceParam()}})
}

func RegisterAnalyticsRoutes(api RouteRegistrar, h *AnalyticsHandler) {
	analytics := api.Group("/analytics", "analytics")
	analytics.Get("/dashboard", h.GetDashboard, RouteMeta{Summary: "Dashboard summary: balance sheet, month-over-month change, 30-day net worth trend, cash flow history and top accounts.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/budget", h.GetBudgetReport, RouteMeta{Summary: "Budget actuals for a month: spent, remaining and status per category, with burn rate and, for the current month, burnOutDate (yyyy-mm-dd the limit runs out at the current pace; null when it will not this month).", QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "YYYY-MM; defaults to the current month.")}})
	analytics.Get("/budget/history", h.GetBudgetHistory, RouteMeta{Summary: "Limit and spending per expense category for the months ending at month, oldest first: {months:[YYYY-MM], rows:[{category, values:[{month, limit, spent}]}]}.", QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "Last month, YYYY-MM; defaults to the current month."), QueryParam("months", "integer", "Number of months, 1-24.", defaultBudgetHistory)}})
	analytics.Get("/cash-flow/yearly", h.GetYearlyCashFlow, RouteMeta{Summary: "Income and expense of a calendar year: per month (January first) with cash flow and savings rate, year total, monthly average over the elapsed months, the same months of the previous year with % change, per category per month (largest first), and the years that have data (newest first).", QueryParams: []RouteParameter{forceParam(), QueryParam("year", "integer", "Calendar year; defaults to the current year.")}})
	analytics.Get("/cards", h.GetCards, RouteMeta{Summary: "Credit cards per bank: the shared limit, usage (what the billed accounts owe) and remaining room; per card the current period's standing (spend, confirmed and estimated cashback, caps left, next tier), the pay-by date and 36 periods of history reconciled with the refunds received.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/cards/suggest", h.SuggestCards, RouteMeta{Summary: "Ranks the held cards for a payment by the cashback it would add to each card's current period, cards whose bank has room for it first. The MCC comes from the mcc parameter (confirmed), else the payee's Merchants mapping, else it is unknown and only fallback rules apply.", QueryParams: []RouteParameter{forceParam(), QueryParam("payee", "string", "Payee of the payment; required without mcc."), QueryParam("amount", "number", "Payment amount in VND, positive."), QueryParam("mcc", "integer", "Merchant category code, when known.")}})
	analytics.Get("/cards/compare", h.ComparePrograms, RouteMeta{Summary: "Replays the held cards' payments of the last 12 months under every programme's current terms, held or not, ranked by cashback less the annual fee.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/health", h.GetFinancialHealth, RouteMeta{Summary: "Financial health score with per-pillar formulas and recommendations. Debt-to-income is monthly debt payments / average monthly income (reference 35%). projection gives net worth in 5, 10 and 20 years, nominal and inflation-adjusted (real).", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/account-histories", h.GetAccountHistories, RouteMeta{Summary: "Back-calculated daily balances per account.", QueryParams: []RouteParameter{forceParam(), QueryParam("days", "integer", "Number of days, 1-366.", defaultHistoryDays)}})
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
}
