package fiber

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
	accounts.Get("/", h.GetAccounts, RouteMeta{Summary: "Lists live accounts (soft-deleted rows excluded) with type, balance and currency.", QueryParams: []RouteParameter{forceParam()}})
	accounts.Post("/", guard.Wrap(h.CreateAccount), RouteMeta{
		Summary:     "Creates an account row. Currency defaults to USDT for Binance accounts, else VND, and is stored in column I.",
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
	transactions.Get("/", h.GetTransactions, RouteMeta{Summary: "Lists transactions newest first with categoryType, etag and data issues.", QueryParams: []RouteParameter{forceParam()}})
	transactions.Post("/", guard.Wrap(h.CreateTransaction), RouteMeta{
		Summary:     "Appends a transaction and returns its id. Requires accountName, date, payee, category and payment or deposit.",
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

	api.Group("/categories", "categories").Get("/", h.GetCategories, RouteMeta{Summary: "Lists income, expense and non-budget categories.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/goals", "goals").Get("/", h.GetGoals, RouteMeta{Summary: "Lists savings goals with progress and status.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/loans", "loans").Get("/", h.GetLoans, RouteMeta{Summary: "Lists loans and debts with monthly and yearly progress.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/tags", "tags").Get("/", h.GetTags, RouteMeta{Summary: "Lists normalised transaction tags.", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/investments", "investments").Get("/assets", h.GetInvestmentAssets, RouteMeta{Summary: "Returns crypto and fund holdings, raw and typed (symbol, quantity, price, value).", QueryParams: []RouteParameter{forceParam()}})
	api.Group("/sync", "sync").Post("/", guard.Wrap(h.Sync), RouteMeta{Summary: "Invalidates every sheet-derived cache key so the next reads come from the sheet."})
	api.Group("/init", "init").Get("/", h.Init, RouteMeta{Summary: "Warms AI prompts and knowledge from the sheet and returns row counts.", QueryParams: []RouteParameter{forceParam()}})
}

func RegisterAnalyticsRoutes(api RouteRegistrar, h *AnalyticsHandler) {
	analytics := api.Group("/analytics", "analytics")
	analytics.Get("/dashboard", h.GetDashboard, RouteMeta{Summary: "Dashboard summary: balance sheet, month-over-month change, 30-day net worth trend, cash flow history and top accounts.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/budget", h.GetBudgetReport, RouteMeta{Summary: "Budget actuals for a month: spent, remaining and status per category, with burn rate.", QueryParams: []RouteParameter{forceParam(), QueryParam("month", "string", "YYYY-MM; defaults to the current month.")}})
	analytics.Get("/cards", h.GetCards, RouteMeta{Summary: "Credit card cashback, cycle, utilisation and 36-month history grouped by bank and account.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/health", h.GetFinancialHealth, RouteMeta{Summary: "Financial health score with per-pillar formulas and recommendations.", QueryParams: []RouteParameter{forceParam()}})
	analytics.Get("/account-histories", h.GetAccountHistories, RouteMeta{Summary: "Back-calculated daily balances per account.", QueryParams: []RouteParameter{forceParam(), QueryParam("days", "integer", "Number of days, 1-366.", defaultHistoryDays)}})
}
