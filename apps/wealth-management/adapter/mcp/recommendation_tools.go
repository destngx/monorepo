package mcp

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
)

// Recommendations list tools (ROADMAP R7), matching /api/recommendations.
// Refresh and status changes write only the app-managed Recommendations tab.

// SetRecommendationService exposes the recommendations list as tools.
func (s *Server) SetRecommendationService(recommendations port.RecommendationService) {
	s.recommendations = recommendations
}

func recommendationFilterSchema() map[string]any {
	return map[string]any{
		"status": map[string]any{"type": "string", "enum": []string{"open", "snoozed", "done", "dismissed"}, "description": "Omit for every status."},
		"page":   map[string]any{"type": "string", "description": "App route, e.g. /budget; omit for every page."},
		"source": map[string]any{"type": "string", "description": "Review kind or rule:<id>; omit for every source."},
	}
}

func (s *Server) recommendationTools() []tool {
	recommendations := s.recommendations
	return []tool{
		{
			definition: definition("GetRecommendations", "Lists the recommendations (action, estimated VND impact per year, evidence, page, status open/snoozed/done/dismissed, outcome, etag) with the open count and their total impact. Reads only.", object(withProperties(recommendationFilterSchema(), map[string]any{"force": forceSchema}))),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					domain.RecommendationFilter
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return recommendations.List(ctx, args.RecommendationFilter, args.Force)
			},
		},
		{
			definition: definition("RefreshRecommendations", "Runs the engine rules (rebalance drift, goal plan short, money lent open over 90 days, idle cash worth moving): adds an item while a condition holds and closes it when the condition is gone, then lists like GetRecommendations.", object(recommendationFilterSchema())),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var filter domain.RecommendationFilter
				if err := decodeArguments(arguments, &filter); err != nil {
					return nil, err
				}
				return recommendations.Refresh(ctx, filter)
			},
		},
		{
			definition: definition("SetRecommendationStatus", "Marks a recommendation done, dismissed, snoozed (snoozedUntil yyyy-mm-dd or snoozeDays 1-365) or open again. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
				"id":           stringSchema,
				"etag":         stringSchema,
				"status":       map[string]any{"type": "string", "enum": []string{"open", "done", "dismissed", "snoozed"}},
				"snoozedUntil": map[string]any{"type": "string", "description": "yyyy-mm-dd after today."},
				"snoozeDays":   map[string]any{"type": "integer", "minimum": 1, "maximum": domain.RecommendationMaxSnoozeDays},
			}, "id", "etag", "status")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.RecommendationStatusInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.RecommendationStatusInput.Validate(); err != nil {
					return nil, err
				}
				return recommendations.SetStatus(ctx, args.ID, args.RecommendationStatusInput)
			},
		},
	}
}
