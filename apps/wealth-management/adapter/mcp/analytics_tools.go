package mcp

import (
	"context"
	"encoding/json"
)

func (s *Server) analyticsTools() []tool {
	analytics := s.analyticsService
	return []tool{
		readTool("GetDashboard", "Balance sheet, month-over-month change, 30-day net worth trend, cash flow history and top accounts.", analytics.DashboardSummary),
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
		readTool("GetCardReports", "Credit card cashback, statement cycle, utilisation and history.", analytics.CardReports),
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
