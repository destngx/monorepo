package mcp

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

// Recurring item writes (ROADMAP R4), matching the /api/bills routes. They
// write only the app-managed Recurring tab.

func recurringSchema() map[string]any {
	return map[string]any{
		"name":      stringSchema,
		"kind":      map[string]any{"type": "string", "enum": []string{"income", "bill", "subscription", "insurance", "card"}},
		"amount":    map[string]any{"type": "number", "exclusiveMinimum": 0, "description": "VND. Omit on card rows to use the statement balance."},
		"frequency": map[string]any{"type": "string", "enum": []string{"monthly", "quarterly", "yearly"}, "description": "Default monthly."},
		"day":       map[string]any{"type": "integer", "minimum": 1, "maximum": 31, "description": "Day of month; not needed on card rows."},
		"month":     map[string]any{"type": "integer", "minimum": 1, "maximum": 12, "description": "First month of a quarterly item, month of a yearly one."},
		"account":   stringSchema,
		"category":  stringSchema,
		"match":     map[string]any{"type": "string", "description": "Payee text that marks a transaction as this item."},
		"status":    map[string]any{"type": "string", "enum": []string{"active", "paused"}},
		"note":      stringSchema,
	}
}

func (s *Server) billsWriteTools() []tool {
	db := s.databaseService
	schema := recurringSchema()
	return []tool{
		{
			definition: definition("CreateRecurringItem", "Adds a recurring item (income, bill, subscription, insurance or card) to the Recurring tab, creating the tab on first use.", object(schema, "name", "kind")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.RecurringInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateRecurring(input)
			},
		},
		{
			definition: definition("ConfirmRecurringSuggestion", "Writes a suggestion from GetBills to the Recurring tab (created on first use); given fields override the suggested ones, status paused records a dismissal.", object(map[string]any{
				"key": stringSchema, "name": stringSchema, "kind": schema["kind"], "amount": schema["amount"],
				"day": schema["day"], "account": stringSchema, "category": stringSchema, "match": stringSchema,
				"status": schema["status"], "note": stringSchema,
			}, "key")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.RecurringConfirmInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.ConfirmRecurringSuggestion(input)
			},
		},
		{
			definition: definition("UpdateRecurringItem", "Replaces a recurring item. Fails with a conflict when the row changed since its etag was read.", object(withProperties(recurringSchema(), map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}), "id", "etag", "name", "kind")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.RecurringUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.RecurringUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateRecurring(args.ID, args.RecurringUpdateInput)
			},
		},
		{
			definition: definition("SetRecurringItemStatus", "Pauses or resumes a recurring item. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
				"id": stringSchema, "etag": stringSchema, "status": schema["status"],
			}, "id", "etag", "status")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.RecurringStatusInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.RecurringStatusInput.Validate(); err != nil {
					return nil, err
				}
				return db.SetRecurringStatus(args.ID, args.RecurringStatusInput)
			},
		},
		{
			definition: definition("DeleteRecurringItem", "Clears a recurring item row. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
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
				if err := db.DeleteRecurring(args.ID, args.ETag); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
	}
}
