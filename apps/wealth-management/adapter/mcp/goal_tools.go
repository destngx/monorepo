package mcp

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

// Goal Plan writes (D31), matching the /api/goals routes.

func goalSchema() map[string]any {
	return map[string]any{
		"name":           stringSchema,
		"target":         map[string]any{"type": "number", "exclusiveMinimum": 0, "description": "VND. Omit only on the emergency fund (name or note mentions emergency) to compute it from monthly needs."},
		"targetDate":     map[string]any{"type": "string", "pattern": `^\d{4}-\d{1,2}$`, "description": "yyyy-mm; omit for no deadline."},
		"priority":       map[string]any{"type": "integer", "minimum": 0, "description": "1 is filled first; 0 or omitted keeps it last on create, unchanged on update."},
		"startingAmount": numberSchema,
		"status":         map[string]any{"type": "string", "enum": []string{string(domain.GoalActive), string(domain.GoalPaused), string(domain.GoalDone)}},
		"note":           stringSchema,
	}
}

func (s *Server) goalWriteTools() []tool {
	db := s.databaseService
	return []tool{
		{
			definition: definition("SetupGoalPlan", "First use: creates the Goal Plan tab with an emergency fund row and imports the old Goals rows (the old tab is only read). Fails with a conflict when the plan already has goals.", nil),
			handler: func(context.Context, json.RawMessage) (any, error) {
				return db.SetupGoalPlan()
			},
		},
		{
			definition: definition("CreateGoal", "Adds a goal to the Goal Plan and returns it with its id and etag.", object(goalSchema(), "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.GoalInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateGoal(input)
			},
		},
		{
			definition: definition("UpdateGoal", "Replaces a goal (pause, resume or mark done through status). Fails with a conflict when the row changed since its etag was read.", object(withProperties(goalSchema(), map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}), "id", "etag", "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.GoalUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.GoalUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateGoal(args.ID, args.GoalUpdateInput)
			},
		},
		{
			definition: definition("DeleteGoal", "Clears a goal row. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}, "id", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID   string `json:"id"`
					ETag string `json:"etag"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := db.DeleteGoal(args.ID, args.ETag); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("ReorderGoals", "Sets goal priorities 1..n in the order of ids; goals not listed follow in their current order.", object(map[string]any{
				"ids": map[string]any{"type": "array", "minItems": 1, "items": stringSchema},
			}, "ids")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.GoalReorderInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.ReorderGoals(input)
			},
		},
	}
}
