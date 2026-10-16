package mcp

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

func (s *Server) analyticsTools() []tool {
	analytics := s.analyticsService
	return []tool{
		readTool("GetGoals", "The goal plan: goals are claims on one pooled savings balance (assets without crypto, archived accounts and money lent, minus card debt and a spending buffer) filled by priority, emergency fund first. Per goal target, funded, remaining, required per month, projected finish, years to reach and pace (done, paused, funded, on_track, behind, no_deadline); the plan check (required per month vs average monthly savings, gap, verdict, trade-offs). setupNeeded until the Goal Plan tab exists.", analytics.GoalPlan),
		readTool("GetMoneyLent", "Money lent per person from lending-category transactions: lent, repaid, outstanding (negative = the user owes that person), last activity, days outstanding and the transactions; totals owed to the user and owed by the user.", analytics.MoneyLent),
		readTool("GetDashboard", "Balance sheet, net worth change this month, month-to-date income and expense against the same days of last month, month-end expense pace, 30-day net worth trend, cash flow history and top accounts.", analytics.DashboardSummary),
		{
			definition: definition("GetBudgetReport", "Budget actuals for a month: spent, remaining and status per category.", object(map[string]any{
				"month": map[string]any{"type": "string", "description": "YYYY-MM; defaults to the current month."}, "force": forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Month string `json:"month"`
					Force bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.BudgetReport(args.Month, args.Force)
			},
		},
		{
			definition: definition("GetBudgetHistory", "Limit and spend per expense category for several months ending at month, oldest first.", object(map[string]any{
				"month":  map[string]any{"type": "string", "description": "YYYY-MM; defaults to the current month."},
				"months": map[string]any{"type": "integer", "minimum": 1, "maximum": 24, "default": 6},
				"force":  forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				args := struct {
					Month  string `json:"month"`
					Months int    `json:"months"`
					Force  bool   `json:"force"`
				}{Months: 6}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.BudgetHistory(args.Month, args.Months, args.Force)
			},
		},
		{
			definition: definition("GetYearlyCashFlow", "Income and expense of a calendar year per month and per category, with year total, monthly average, and change against the same months of the previous year.", object(map[string]any{
				"year": map[string]any{"type": "integer", "description": "Calendar year; defaults to the current year."}, "force": forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Year  int  `json:"year"`
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.YearlyCashFlow(args.Year, args.Force)
			},
		},
		readTool("GetCardReports", "Credit cards per bank: shared limit, usage and room; per card the current period's spend, confirmed and estimated cashback, caps left, next tier, pay-by date and reconciled history.", analytics.CardReports),
		{
			definition: definition("SuggestCardForPayment", "Ranks the held cards for a payment by the cashback it would add this period, cards whose bank has room first. Pass mcc when known; otherwise the payee's mapping is used, else only fallback rules apply.", object(map[string]any{
				"payee":  map[string]any{"type": "string", "description": "Payee of the payment; required without mcc."},
				"amount": map[string]any{"type": "number", "exclusiveMinimum": 0, "description": "Payment amount in VND."},
				"mcc":    map[string]any{"type": "integer", "minimum": 0, "maximum": 9999, "description": "Merchant category code, when known."},
				"force":  forceSchema,
			}, "amount")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Payee  string  `json:"payee"`
					Amount float64 `json:"amount"`
					MCC    *int    `json:"mcc"`
					Force  bool    `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.SuggestCards(domain.CardPaymentQuery{Payee: args.Payee, Amount: args.Amount, MCC: args.MCC}, args.Force)
			},
		},
		readTool("CompareCardPrograms", "Replays the last 12 months of card payments under every programme's current terms, held or not, ranked by cashback less the annual fee.", analytics.ComparePrograms),
		readTool("GetFinancialHealth", "Financial health score with per-pillar formulas and recommendations.", analytics.FinancialHealth),
		{
			definition: definition("GetAccountHistories", "Back-calculated daily balances per account.", object(map[string]any{
				"days": map[string]any{"type": "integer", "minimum": 1, "maximum": 366, "default": 60}, "force": forceSchema,
			})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				args := struct {
					Days  int  `json:"days"`
					Force bool `json:"force"`
				}{Days: 60}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.AccountHistories(args.Days, args.Force)
			},
		},
	}
}

// readTool wraps a cached list call whose only argument is force.
func readTool[T any](name, description string, read func(forceFresh bool) (T, error)) tool {
	return tool{
		definition: definition(name, description, object(map[string]any{"force": forceSchema})),
		handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
			var args struct {
				Force bool `json:"force"`
			}
			if err := decodeArguments(arguments, &args); err != nil {
				return nil, err
			}
			return read(args.Force)
		},
	}
}
