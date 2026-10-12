package fiber

// Route registration for the AI use cases (§10). Every model call costs
// money and may be slow, so all of them sit behind the WriteGuard.

func aiOptionProperties(extra map[string]any) map[string]any {
	properties := map[string]any{
		"force": map[string]any{"type": "boolean", "description": "Bypass the 30-minute result cache."},
		"model": map[string]any{"type": "string", "description": "Model override for this request."},
	}
	for key, value := range extra {
		properties[key] = value
	}
	return properties
}

func optionalJSONBody(properties map[string]any) map[string]any {
	body := jsonBody(properties)
	body["required"] = false
	return body
}

func RegisterAIRoutes(api RouteRegistrar, h *AIHandler, guard WriteGuard) {
	ai := api.Group("/ai", "ai")
	ai.Get("/status", h.GetStatus, RouteMeta{Summary: "AI configuration (endpoint host, models, fallback) and required prompts missing from the Prompts sheet."})
	ai.Post("/suggest-category", guard.Wrap(h.SuggestCategory), RouteMeta{
		Summary:     "Suggests a category for a payee: the payee's own history first, then the model, constrained to known categories.",
		RequestBody: jsonBody(map[string]any{"payee": schemaString, "model": schemaString}, "payee"),
	})
	ai.Post("/review", guard.Wrap(h.Review), RouteMeta{
		Summary: "Structured review (summary, sections, action items) of accounts, budget, loans, the last 7 days of transactions, or credit cards.",
		RequestBody: jsonBody(aiOptionProperties(map[string]any{
			"kind":  map[string]any{"type": "string", "enum": []string{"account-review", "budget-review", "loan-review", "transaction-review", "credit-summary"}},
			"month": map[string]any{"type": "string", "description": "YYYY-MM for budget and credit-card reviews; defaults to the current month."},
			"card":  map[string]any{"type": "string", "description": "Credit-card account name; empty reviews every card."},
		}), "kind"),
	})
	ai.Post("/health-commentary", guard.Wrap(h.HealthCommentary), RouteMeta{
		Summary:     "Structured commentary on the deterministic financial health score.",
		RequestBody: optionalJSONBody(aiOptionProperties(nil)),
	})
	ai.Post("/chart-insight", guard.Wrap(h.ChartInsight), RouteMeta{
		Summary: "Markdown commentary on the data behind a chart (at most 12 KB of data).",
		RequestBody: jsonBody(aiOptionProperties(map[string]any{
			"chartType":   schemaString,
			"title":       schemaString,
			"description": schemaString,
			"timeframe":   schemaString,
			"data":        map[string]any{"description": "The series as rendered."},
		}), "chartType", "data"),
	})
	ai.Post("/briefing", guard.Wrap(h.Briefing), RouteMeta{
		Summary:     "Dashboard intelligence briefing: markdown briefing, alerts and pattern cards, with the engine's current-month facts.",
		RequestBody: optionalJSONBody(aiOptionProperties(nil)),
	})
	ai.Post("/budget-advisor", guard.Wrap(h.BudgetAdvice), RouteMeta{
		Summary:     "Budget advisor: suggested limits and alerts from the model; averages, velocity flags, 30-day projection and forecasted savings from the engine.",
		RequestBody: optionalJSONBody(aiOptionProperties(nil)),
	})
	ai.Post("/chat", guard.Wrap(h.Chat), RouteMeta{
		Summary: "Tool-using chat streamed as NDJSON events: text (delta), tool (name, status), done (model) and error (message).",
		RequestBody: jsonBody(map[string]any{
			"messages": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"role": map[string]any{"type": "string", "enum": []string{"user", "assistant"}}, "content": schemaString}}},
			"page":     map[string]any{"type": "object", "properties": map[string]any{"path": schemaString, "title": schemaString, "summary": schemaString}},
			"model":    schemaString,
		}, "messages"),
	})
}
