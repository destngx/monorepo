package mcp

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

// Writes for the inputs that were sheet-only before AI form filling (D46),
// matching the /api/investments and /api/cards write routes.

func cardSchema() map[string]any {
	return map[string]any{
		"name":         stringSchema,
		"program":      map[string]any{"type": "string", "description": "A programme in the Programs tab."},
		"keyword":      map[string]any{"type": "string", "description": "Tag word that marks a payment as made with this card."},
		"expiry":       map[string]any{"type": "string", "description": "As printed, e.g. 08/29."},
		"statementDay": map[string]any{"type": "integer", "minimum": 0, "maximum": 31},
		"dueDay":       map[string]any{"type": "integer", "minimum": 0, "maximum": 31},
		"default":      map[string]any{"type": "boolean"},
		"closedFrom":   map[string]any{"type": "string", "description": "yyyy-mm-dd; omit while the card is open."},
		"last4":        stringSchema,
		"limit":        map[string]any{"type": "number", "minimum": 0, "description": "VND credit limit of the bank's shared account; omit to leave it."},
	}
}

func (s *Server) configWriteTools() []tool {
	db := s.databaseService
	return []tool{
		{
			definition: definition("RecordFundTrade", "Records a fund certificate trade: appends it to the fund's History and adds the units to Total Unit. Negative units are a sale; a code not held gets a new row.", object(map[string]any{
				"code":  stringSchema,
				"date":  map[string]any{"type": "string", "description": "yyyy-mm-dd, not in the future."},
				"units": map[string]any{"type": "number", "description": "Units bought; negative for units sold."},
			}, "code", "date", "units")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.FundTradeInput
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return db.RecordFundTrade(input)
			},
		},
		{
			definition: definition("SetCryptoHoldings", "Sets how much of each coin a platform holds now in the Crypto tab, adding rows for coins not listed.", object(map[string]any{
				"holdings": map[string]any{"type": "array", "minItems": 1, "maxItems": 50, "items": object(map[string]any{
					"platform": stringSchema,
					"symbol":   map[string]any{"type": "string", "description": "Ticker, e.g. BTC."},
					"quantity": numberSchema,
					"note":     stringSchema,
				}, "platform", "symbol", "quantity")},
			}, "holdings")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.CryptoHoldingsInput
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				holdings, err := db.SetCryptoHoldings(input)
				if err != nil {
					return nil, err
				}
				return map[string]any{"holdings": holdings}, nil
			},
		},
		{
			definition: definition("CreateCard", "Adds a credit card to the Cards tab, and sets the bank's shared credit limit when limit is given. Fails when the result would be an invalid card row.", object(cardSchema(), "name", "program")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.CardInput
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateCard(input)
			},
		},
		{
			definition: definition("UpdateCard", "Replaces a card's Cards row; card is its current name.", object(withProperties(cardSchema(), map[string]any{"card": stringSchema}), "card", "name", "program")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Card string `json:"card"`
					domain.CardInput
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return db.UpdateCard(args.Card, args.CardInput)
			},
		},
		{
			definition: definition("CreateCardProgram", "Adds a cashback programme version with its rules (Programs and ProgramRules tabs). Fails with a conflict when the programme already has terms from that date.", object(map[string]any{
				"program":       stringSchema,
				"bank":          stringSchema,
				"sharedAccount": map[string]any{"type": "string", "description": "For a bank with no Banks row: the account its cards bill to; adds the row."},
				"from":          map[string]any{"type": "string", "description": "yyyy-mm-dd the terms start."},
				"annualFee":     numberSchema,
				"cap":           map[string]any{"type": "number", "minimum": 0, "description": "Card-wide cashback cap per cap period."},
				"capPeriod":     map[string]any{"type": "string", "enum": []string{string(domain.CapPeriodCycle), string(domain.CapPeriodMonth)}},
				"rules": map[string]any{"type": "array", "minItems": 1, "maxItems": 50, "items": object(map[string]any{
					"rule":           stringSchema,
					"mccs":           map[string]any{"type": "string", "description": "MCC codes and ranges, e.g. 5411,5812-5814; omit to match any."},
					"rate":           map[string]any{"type": "number", "description": "Cashback percent."},
					"cap":            numberSchema,
					"minTransaction": numberSchema,
					"minCycleSpend":  numberSchema,
					"fallback":       map[string]any{"type": "boolean"},
					"exclude":        map[string]any{"type": "boolean"},
					"capGroup":       stringSchema,
				}, "rule", "rate")},
			}, "program", "bank", "from", "capPeriod", "rules")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.ProgramInput
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return db.CreateCardProgram(input)
			},
		},
		{
			definition: definition("SaveMerchant", "Saves the MCC a payee pattern is billed with (Merchants tab), replacing the row with the same pattern and start date.", object(map[string]any{
				"pattern":    stringSchema,
				"mcc":        map[string]any{"type": "integer", "minimum": 1, "maximum": 9999},
				"from":       map[string]any{"type": "string", "description": "yyyy-mm-dd; omit to apply always."},
				"confidence": map[string]any{"type": "string", "enum": []string{string(domain.MCCAssumed), string(domain.MCCConfirmed)}},
				"note":       map[string]any{"type": "string", "description": "Where the MCC came from."},
			}, "pattern", "mcc")),
			handler: func(_ context.Context, arguments json.RawMessage) (any, error) {
				var input domain.MerchantInput
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return db.SaveMerchant(input)
			},
		},
	}
}
