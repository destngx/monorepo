package mcp

import (
	"apps/wealth-management/port"
	"context"
	"encoding/json"
)

// Monthly close and net worth history (ROADMAP R8, P3), matching the
// /api/close and /api/net-worth routes. WriteMonthSnapshot writes only the
// app-managed Snapshots tab.

// SetMonthlyCloseService exposes the monthly close as tools.
func (s *Server) SetMonthlyCloseService(monthlyClose port.MonthlyCloseService) {
	s.monthlyClose = monthlyClose
}

func (s *Server) closeTools() []tool {
	service := s.monthlyClose
	return []tool{
		{
			definition: definition("GetMonthlyClose", "Monthly close checklist for a month (default: last month during the first 10 days of a month, else this month): approve capture drafts, categorize (rows Uncategorized or in an unknown category), budget vs actual (categories over their limit), surplus to goals (the month's income - expense vs what dated goals need a month), chase money lent (open 30+ days), review recommendations (open items, their impact a year) and write the snapshot. Each item has a state (done, action, unavailable with the reason), a detail without amounts, and count, amount and target where they apply.", object(map[string]any{
				"month": map[string]any{"type": "string", "description": "yyyy-mm, up to this month."}, "force": forceSchema,
			})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Month string `json:"month"`
					Force bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return service.MonthlyClose(ctx, args.Month, args.Force)
			},
		},
		{
			definition: definition("WriteMonthSnapshot", "Writes the month's row to the app-managed Snapshots tab (net worth, assets, liabilities, money lent, cash, funds, crypto, income, expense, savings, savings rate, health score and pillars, emergency months), creating the tab on first use. Updates the month's existing row instead of adding one (created false). Only this month (as of today) or last month (balances taken back to its last day).", object(map[string]any{
				"month": map[string]any{"type": "string", "description": "yyyy-mm: this month or last month; blank = the close month."},
			})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Month string `json:"month"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return service.WriteSnapshot(ctx, args.Month)
			},
		},
		{
			definition: definition("GetNetWorthHistory", "Net worth per month, oldest first: snapshot months from the Snapshots tab, other past months estimated (back-calculated from transactions, missing price moves), this month live. Each change is split into savings (income - expense) and market (the rest) when both months are measured; milestones are the first crossing of round levels (1-2-5 steps from 100 million VND) and the all-time high.", object(map[string]any{"force": forceSchema})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return service.NetWorthHistory(ctx, args.Force)
			},
		},
	}
}
