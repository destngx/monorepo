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
		readTool("GetBills", "Bills and safe-to-spend: the Recurring tab items (income, bill, subscription, insurance, card) with this cycle's occurrence settled by matching transactions (payee contains the item's match text; category and account agree when set), suggestions found in history (confirm by key with ConfirmRecurringSuggestion), the calendar from the 1st of this month to 30 days ahead (paid, due, overdue; card payments due on the pay-by date for their statement) and safeToSpend = ready money (active, rarely used, bank and cash accounts without crypto or accounts marked not ready to use; excludedAccounts gives the reason) - committed (unpaid bills and card payments due before the next income, overdue ones too, one due on payday being paid by that wage, else in the next 30 days inclusive; plus the fixed costs still to pay: fixed categories' monthly budget, else 6-month average, minus spent and their unpaid recurring bills) - essentials (essential categories' average daily spending, else budget, times the days) = free to spend, floored at 0; shortBy when committed is not covered, tightBy when essentials are not; no goal money is subtracted.", analytics.Bills),
		readTool("GetSafeToSpend", "Safe-to-spend until the next wage, in tiers: ready money (without crypto or accounts marked not ready) minus committed (unpaid bills and card payments due before payday, overdue ones too, and fixed costs still to pay) minus essentials until payday = free to spend (safeToSpend), with every component, the bills facts (unpaid in the next 30 days, overdue count and amount), overdue bills and unpaid bills due within 7 days.", func(force bool) (domain.BillsSummary, error) {
			bills, err := analytics.Bills(force)
			if err != nil {
				return domain.BillsSummary{}, err
			}
			return bills.Summary(), nil
		}),
		{
			definition: definition("GetIdleCash", "Idle cash check (D34): idle = max(0, readyMoney - keepLiquid - dayToDayFloat). readyMoney is safe-to-spend's ready money (active, rarely used, bank and cash accounts; no crypto, no accounts marked not ready) without the emergency accounts (idleCash.emergencyAccounts). keepLiquid = unpaid bills and card payments in the next 30 days (overdue too) + this month's goal set-aside + the part of the emergency fund target the emergency accounts do not cover. dayToDayFloat = idleCash.dayToDayFloat, else the spending buffer (buffer months x average monthly expense). The idle part's weighted current rate is read from account notes ('earn rate 4%', 'lãi 4%'; else 0; lowest-yield accounts first). best is the best instant-access rate only: a no-term deposit (withdraw any time), a 1-month deposit (rolls monthly; early withdrawal pays the no-term rate) or a flexible savings product such as an e-wallet earning balance (access flexible, backing deposit or business_cooperation = not a bank deposit), with access no_term, one_month or flexible; longer terms and rates without a tenor are never used. options lists every place compared, best first. An account linked to a product (LinkAccountProduct) earns its live rate. moves (idle part to best), emergency.moves and switches (a linked wallet account's day-to-day money to a better flexible product) price each move: extra a month and a year, the days without interest and interest lost moving today and on the cheapest day of the next week (products pay from the next working day, so a Friday move loses 3 days), the payback days and the steps. ladder splits idle cash for a 1-month best into 2-3 rolling parts. Verdict: worth_moving when the gap is at least 0.5 points and 200.000 VND a year; not_worth_it, nothing_idle, no_rate (no instant-access rate) with reasons. emergency: the emergency accounts, their rate, and betterRate when the best instant-access rate beats them by the same thresholds. About rate only: goal money stays pooled.", object(map[string]any{"force": forceSchema})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return analytics.IdleCash(ctx, args.Force)
			},
		},
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
		readTool("GetSpendingRhythm", "How spending is going (WM-BUD-12): today against a typical day, the last 30 days and the last 3 months (90 days) ending today, each with spent = fixed (categories tagged fixed, paid once) + dayToDay, dailyAverage (dayToDay / days), monthlyAverage, the previous period of the same length and expense categories with split tags; daily is the last 90 days. recommendations: faster or slower (30-day daily average against the 3-month one, at 15%), category_rising and split_rising (the last 30 days more than 25% above the usual 30 days and at least 5% of the 30 days' day-to-day spend; fixed categories never, essentials in the info tone: look at what changed, do not cut a need), heavy_day (today at least twice a typical day).", analytics.SpendingWindows),
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
		readTool("GetFinancialHealth", "Financial health score: eight pillars with their formulas, the strict caps that hold the score down, and recommendations.", analytics.FinancialHealth),
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
