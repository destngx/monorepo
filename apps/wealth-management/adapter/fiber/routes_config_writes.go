package fiber

// Routes for the inputs that were sheet-only before AI form filling (D47,
// WM-FILL-10..14). Writes go through the WriteGuard.

func cardProperties() map[string]any {
	return map[string]any{
		"name":         schemaString,
		"program":      map[string]any{"type": "string", "description": "A programme in the Programs tab."},
		"keyword":      map[string]any{"type": "string", "description": "Tag word that marks a payment as made with this card."},
		"expiry":       map[string]any{"type": "string", "description": "As printed, e.g. 08/29; kept as text."},
		"statementDay": map[string]any{"type": "integer", "minimum": 0, "maximum": 31},
		"dueDay":       map[string]any{"type": "integer", "minimum": 0, "maximum": 31},
		"default":      schemaBoolean,
		"closedFrom":   map[string]any{"type": "string", "description": "yyyy-mm-dd; blank while the card is open."},
		"last4":        schemaString,
		"limit":        map[string]any{"type": "number", "description": "VND credit limit of the bank's shared account (Accounts C); omit to leave it."},
	}
}

func programRuleProperties() map[string]any {
	return map[string]any{
		"rule":           schemaString,
		"mccs":           map[string]any{"type": "string", "description": "MCC codes and ranges, e.g. 5411,5812-5814; blank matches any."},
		"rate":           map[string]any{"type": "number", "description": "Cashback percent."},
		"cap":            schemaNumber,
		"minTransaction": schemaNumber,
		"minCycleSpend":  schemaNumber,
		"fallback":       schemaBoolean,
		"exclude":        schemaBoolean,
		"capGroup":       schemaString,
	}
}

func registerConfigWriteRoutes(api RouteRegistrar, h *DatabaseHandler, guard WriteGuard) {
	investments := api.Group("/investments", "investments")
	investments.Post("/funds/trades", guard.Wrap(h.RecordFundTrade), RouteMeta{
		Summary: "Records a fund certificate trade: appends d-m-yyyy,units to the fund's History cell and adds the units to Total Unit in one write (InvestmentFundCertificate). Negative units are a sale. A code not held gets a row in the first blank row of the table. Returns {code, units (held after), entry, row, created}; 400 for a future date, zero units or selling more than held.",
		RequestBody: jsonBody(map[string]any{
			"code":  schemaString,
			"date":  map[string]any{"type": "string", "description": "yyyy-mm-dd, not in the future."},
			"units": map[string]any{"type": "number", "description": "Units bought; negative for units sold. Rounded to 2 decimals."},
		}, "code", "date", "units"),
	})
	investments.Post("/crypto/holdings", guard.Wrap(h.SetCryptoHoldings), RouteMeta{
		Summary: "Sets how much of each coin a platform holds now in the Crypto tab: updates the Amount of a listed platform and token, or adds a row. Creates the tab, or a Platform|Token|Amount|Note header below its rows, when missing. Returns {holdings:[{platform, symbol, quantity, row, created}]}; 400 for a coin listed twice for one platform.",
		RequestBody: jsonBody(map[string]any{"holdings": map[string]any{"type": "array", "maxItems": 50, "items": map[string]any{"type": "object", "properties": map[string]any{
			"platform": schemaString,
			"symbol":   schemaString,
			"quantity": schemaNumber,
			"note":     schemaString,
		}, "required": []string{"platform", "symbol", "quantity"}}}}, "holdings"),
	})

	cards := api.Group("/cards", "cards")
	cards.Post("/", guard.Wrap(h.CreateCard), RouteMeta{
		Summary:     "Adds a Cards row (and the bank's shared account limit when limit is set). The tabs are read as they would be after the write; 400 naming the row when the result would be invalid, e.g. an unknown programme or a second default card.",
		RequestBody: jsonBody(cardProperties(), "name", "program"),
	})
	cards.Patch("/:name", guard.Wrap(h.UpdateCard), RouteMeta{
		Summary:     "Replaces the card's Cards row (and the shared account limit when limit is set). 404 for an unknown card; 400 when the result would be invalid.",
		PathParams:  []RouteParameter{PathParam("name", "string", "Card name.")},
		RequestBody: jsonBody(cardProperties(), "name", "program"),
	})
	cards.Post("/programs", guard.Wrap(h.CreateCardProgram), RouteMeta{
		Summary: "Adds a programme version (a Programs row) and its rules (ProgramRules rows), and a Banks row when sharedAccount is given for a bank without one. 409 when the programme already has a version from that date; 400 when the result would be invalid.",
		RequestBody: jsonBody(map[string]any{
			"program":       schemaString,
			"bank":          schemaString,
			"sharedAccount": map[string]any{"type": "string", "description": "For a bank with no Banks row: the account its cards bill to; adds the row."},
			"from":          map[string]any{"type": "string", "description": "yyyy-mm-dd the terms start."},
			"annualFee":     schemaNumber,
			"cap":           map[string]any{"type": "number", "description": "Card-wide cashback cap per cap period."},
			"capPeriod":     map[string]any{"type": "string", "enum": []string{"cycle", "month"}, "description": "cycle: each statement cycle; month: each calendar month."},
			"rules":         map[string]any{"type": "array", "minItems": 1, "maxItems": 50, "items": map[string]any{"type": "object", "properties": programRuleProperties(), "required": []string{"rule", "rate"}}},
		}, "program", "bank", "from", "rules"),
	})
	cards.Post("/merchants", guard.Wrap(h.SaveMerchant), RouteMeta{
		Summary: "Saves the MCC a payee pattern is billed with (Merchants tab): replaces the row with the same pattern and start date, else adds one. Returns the mapping.",
		RequestBody: jsonBody(map[string]any{
			"pattern":    map[string]any{"type": "string", "description": "Matched against payees, case and accents ignored."},
			"mcc":        map[string]any{"type": "integer", "minimum": 1, "maximum": 9999},
			"from":       map[string]any{"type": "string", "description": "yyyy-mm-dd; blank applies always."},
			"confidence": map[string]any{"type": "string", "enum": []string{"assumed", "confirmed"}, "description": "Default assumed."},
			"note":       map[string]any{"type": "string", "description": "Where the MCC came from."},
		}, "pattern", "mcc"),
	})
}
