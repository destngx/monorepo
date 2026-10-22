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
			definition: definition("SetAccountReadyToUse", "Marks an account ready to use or not for safe-to-spend: ready false leaves it out of ready money (slow to withdraw) by adding it to safeToSpend.notReadyAccounts in the Preferences tab (created on first use); ready true removes it, also for a name that is no longer an account. etag is safeToSpend.readiness.etag from GetBills or GetSafeToSpend; fails with a conflict when the setting changed since.", object(map[string]any{
				"account": stringSchema, "ready": map[string]any{"type": "boolean"}, "etag": stringSchema,
			}, "account", "ready", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.AccountReadinessInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.SetAccountReadiness(input)
			},
		},
		{
			definition: definition("SetEmergencyAccount", "Marks an account as an emergency fund account or not for the idle cash check: emergency true adds it to idleCash.emergencyAccounts in the Preferences tab (created on first use), so its balance covers the emergency fund target first and it is never called idle; false removes it, also for a name that is no longer an account. etag is emergency.settings.etag from GetIdleCash (or idleCash.etag from GetPreferences); fails with a conflict when the setting changed since.", object(map[string]any{
				"account": stringSchema, "emergency": map[string]any{"type": "boolean"}, "etag": stringSchema,
			}, "account", "emergency", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.EmergencyAccountInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.SetEmergencyAccount(input)
			},
		},
		{
			definition: definition("LinkAccountProduct", "Links an account to the flexible savings product it holds (an e-wallet earning balance such as Zalopay Số dư sinh lời or MoMo Túi Thần Tài): product is an id from GetFlexibleSavings (e.g. zalopay:so-du-sinh-loi), saved in idleCash.accountProducts in the Preferences tab (created on first use). GetIdleCash then uses the product's live rate as the account's current rate and suggests switching its day-to-day money when another flexible product pays materially more. A blank product unlinks it, also for a name that is no longer an account. etag is emergency.settings.accountProducts.etag from GetIdleCash (or idleCash.accountProducts.etag from GetPreferences); fails with a conflict when the setting changed since.", object(map[string]any{
				"account": stringSchema,
				"product": map[string]any{"type": "string", "description": "Product id; blank unlinks."},
				"etag":    stringSchema,
			}, "account", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.AccountProductInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.SetAccountProduct(input)
			},
		},
		{
			definition: definition("SetAccountMinimumBalance", "Sets or removes an account's minimum balance to keep, e.g. the average monthly balance a bank asks for a reward: minimum (VND) and an optional note are saved in accounts.minimumBalance in the Preferences tab (created on first use). That money is never counted as ready to spend or idle, and GetBills tracks the month-to-date average against it (safeToSpend.minimumBalances). null or 0 removes the minimum, also for a name that is no longer an account. etag is safeToSpend.minimumBalance.etag from GetBills or GetSafeToSpend (or accountMinimums.etag from GetPreferences); fails with a conflict when the setting changed since.", object(map[string]any{
				"account": stringSchema,
				"minimum": map[string]any{"type": []string{"number", "null"}, "minimum": 0, "description": "VND to keep; null or 0 removes it."},
				"note":    map[string]any{"type": "string", "maxLength": 200, "description": "Why it is kept, e.g. monthly points reward."},
				"etag":    stringSchema,
			}, "account", "etag")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.AccountMinimumInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.SetAccountMinimumBalance(input)
			},
		},
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
