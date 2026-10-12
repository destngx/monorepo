package mcp

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

func (s *Server) databaseTools() []tool {
	db := s.databaseService
	return []tool{
		readTool("GetAccounts", "Lists live accounts with type, balance, cleared balance and currency.", db.ListAccounts),
		{
			definition: definition("CreateAccount", "Creates an account. Currency defaults to USDT for Binance accounts, else VND.", object(map[string]any{
				"name": stringSchema, "type": accountTypeSchema, "balance": numberSchema, "currency": currencySchema, "note": stringSchema,
			}, "name", "type")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.AccountCreateInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				if err := db.CreateAccount(input); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("UpdateAccount", "Edits an account's type and/or note. Balances come from transactions and cannot be edited.", object(map[string]any{
				"name": stringSchema, "type": accountTypeSchema, "note": stringSchema,
			}, "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Name string `json:"name"`
					domain.AccountUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.AccountUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateAccount(args.Name, args.AccountUpdateInput)
			},
		},
		{
			definition: definition("DeleteAccount", "Soft-deletes an account (its name gets the [DELETED] prefix).", object(map[string]any{"name": stringSchema}, "name")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Name string `json:"name"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := db.DeleteAccount(args.Name); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		readTool("GetTransactions", "Lists transactions newest first. Each has an id (row-N) and an etag needed to update or delete it.", db.ListTransactions),
		{
			definition: definition("CreateTransaction", "Appends a transaction and returns its id. Needs payment or deposit.", object(transactionSchema(), "accountName", "date", "payee", "category")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.TransactionCreateInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateTransaction(input)
			},
		},
		{
			definition: definition("CreateTransactions", "Appends up to 100 transactions in one write. Retrying with the same batchId returns the first result without writing again.", object(map[string]any{
				"batchId":      stringSchema,
				"transactions": map[string]any{"type": "array", "maxItems": domain.MaxTransactionBatch, "items": object(transactionSchema(), "accountName", "date", "payee", "category")},
			}, "batchId", "transactions")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.TransactionBatchInput
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateTransactions(input)
			},
		},
		{
			definition: definition("UpdateTransaction", "Replaces a transaction. Fails with a conflict when the row changed since its etag was read.", object(withProperties(transactionSchema(), map[string]any{
				"id": stringSchema, "etag": stringSchema,
			}), "id", "etag", "accountName", "date", "payee", "category")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					ID string `json:"id"`
					domain.TransactionUpdateInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				if err := args.TransactionUpdateInput.Validate(); err != nil {
					return nil, err
				}
				return db.UpdateTransaction(args.ID, args.TransactionUpdateInput)
			},
		},
		{
			definition: definition("DeleteTransaction", "Clears a transaction row. Fails with a conflict when the row changed since its etag was read.", object(map[string]any{
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
				if err := db.DeleteTransaction(args.ID, args.ETag); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("GetBudget", "Reads a year's budget limits per category (defaults to the current year).", object(map[string]any{"year": yearSchema, "force": forceSchema})),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Year  int  `json:"year"`
					Force bool `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return db.ListBudget(args.Year, args.Force)
			},
		},
		{
			definition: definition("UpdateBudgetLimit", "Sets a category's monthly limit (month = YYYY-MM) or, without month, its yearly limit.", object(map[string]any{
				"year": yearSchema, "category": stringSchema, "month": map[string]any{"type": "string", "pattern": `^\d{4}-\d{2}$`}, "limit": numberSchema,
			}, "category", "limit")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.BudgetLimitUpdate
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				if err := db.UpdateBudgetLimit(input); err != nil {
					return nil, err
				}
				return map[string]any{"success": true}, nil
			},
		},
		{
			definition: definition("ReallocateBudget", "Moves part of a month's limit from one category to another; the total stays the same.", object(map[string]any{
				"year": yearSchema, "month": map[string]any{"type": "string", "pattern": `^\d{4}-\d{2}$`}, "from": stringSchema, "to": stringSchema, "amount": numberSchema,
			}, "month", "from", "to", "amount")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.BudgetReallocation
				if err := decodeValidated(arguments, &input); err != nil {
					return nil, err
				}
				return db.ReallocateBudget(input)
			},
		},
		readTool("GetCategories", "Lists income, expense and non-budget categories.", db.ListCategories),
		readTool("GetGoals", "Lists savings goals with progress and status.", db.ListGoals),
		readTool("GetLoans", "Lists loans and debts with progress.", db.ListLoans),
		readTool("GetTags", "Lists normalised transaction tags.", db.ListTags),
		readTool("GetInvestmentAssets", "Returns crypto and fund holdings with symbol, quantity, price and value.", db.GetInvestmentAssets),
		readTool("GetCaptureRules", "Lists Smart Capture rules: alias (text in a material -> account) and payee (payee -> category).", db.ListCaptureRules),
		{
			definition: definition("SaveCaptureRules", "Creates or updates Smart Capture rules, keyed by kind and pattern (case and accents ignored).", object(map[string]any{
				"rules": map[string]any{"type": "array", "minItems": 1, "items": object(map[string]any{
					"kind":    map[string]any{"type": "string", "enum": []string{string(domain.CaptureRuleAlias), string(domain.CaptureRulePayee)}},
					"pattern": stringSchema,
					"value":   map[string]any{"type": "string", "description": "Account name for alias rules, category for payee rules."},
				}, "kind", "pattern", "value")},
			}, "rules")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Rules []domain.CaptureRule `json:"rules"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return db.SaveCaptureRules(args.Rules)
			},
		},
		readTool("WarmAIContent", "Loads AI prompts and knowledge from the sheet and returns row counts.", db.WarmAIContent),
		{
			definition: definition("GetSheetInfo", "Returns the spreadsheet id and title and the time of the last successful sync.", nil),
			handler: func(context.Context, json.RawMessage) (any, error) {
				return db.SheetInfo()
			},
		},
		{
			definition: definition("SyncSheetsCache", "Invalidates every sheet-derived cache key so the next reads come from the sheet.", nil),
			handler: func(context.Context, json.RawMessage) (any, error) {
				return db.SyncCache()
			},
		},
	}
}
