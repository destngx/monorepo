package mcp

import (
	"apps/wealth-management/port"
	"context"
	"encoding/json"
)

// Home inbox tool (ROADMAP R5), matching GET /api/home/needs-you.

// SetHomeService exposes the home inbox as a tool.
func (s *Server) SetHomeService(home port.HomeService) {
	s.home = home
}

func (s *Server) homeTools() []tool {
	home := s.home
	return []tool{{
		definition: definition("GetNeedsYou", "Lists what needs the user now, ranked by severity (critical, serious, warning, info), "+
			"then estimated yearly impact or amount, then due date: drafts to approve, overdue bills, card payments due within 7 days, "+
			"budget categories over pace, money lent overdue, goal plan short or goals behind, rebalance drift, idle cash, "+
			"uncategorized transactions, last month's close while its snapshot is not written and the top open review recommendations. Each item has kind, status, headline, detail, "+
			"amountVnd, impactVnd, dueDate, count and the app route to act on; unavailable names sources that failed to load. "+
			"Reads only; headlines carry no names of people or goals.", object(map[string]any{"force": forceSchema})),
		handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
			var args struct {
				Force bool `json:"force"`
			}
			if err := decodeArguments(arguments, &args); err != nil {
				return nil, err
			}
			return home.NeedsYou(ctx, args.Force)
		},
	}}
}
