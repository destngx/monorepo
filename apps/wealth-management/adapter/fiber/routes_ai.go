package fiber

import "apps/wealth-management/domain"

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

// reviewRecordSummary describes how a review feeds the recommendations list.
const reviewRecordSummary = "The model sees this review's earlier recommendations (pending, and done in the last 60 days) and may " +
	"return recommendations (0-3: title, action, impactVnd per year or null, evidence) and outcomes (id, verdict met|not_met|unclear, note). " +
	"recorded reports what was written to the Recommendations tab: created (proposals not already pending for the source by title), " +
	"duplicates, checked (items whose outcome was written; a met item still pending becomes done), ignoredOutcomes, and error when the " +
	"list could not be written (the review still stands)."

func RegisterAIRoutes(api RouteRegistrar, h *AIHandler, guard WriteGuard) {
	ai := api.Group("/ai", "ai")
	ai.Get("/status", h.GetStatus, RouteMeta{Summary: "AI configuration (endpoint host, models, fallback) and required prompts missing from the Prompts sheet."})
	ai.Get("/usage", h.GetUsage, RouteMeta{Summary: "OpenAI quota from the ai-gateway GET /v1/usage: rate-limit windows (used percent, reset time) for OAuth accounts, token totals for API-key accounts. 503 UNAVAILABLE when the gateway or provider cannot answer."})
	ai.Get("/models", h.GetModels, RouteMeta{Summary: "Model ids the AI endpoint offers (GET {AI_BASE_URL}/models), sorted; Settings offers them for the model settings. 503 UNAVAILABLE when the endpoint cannot answer."})
	ai.Post("/suggest-category", guard.Wrap(h.SuggestCategory), RouteMeta{
		Summary:     "Suggests a category for a payee: a confirmed CaptureRules payee rule first, then the payee's own history, then the model, constrained to known categories.",
		RequestBody: jsonBody(map[string]any{"payee": schemaString, "model": schemaString}, "payee"),
	})
	ai.Post("/review", guard.Wrap(h.Review), RouteMeta{
		Summary: "Structured review (summary, sections, action items) of accounts, budget, the last 7 days of transactions, credit cards, or a calendar year of income and expense. " + reviewRecordSummary,
		RequestBody: jsonBody(aiOptionProperties(map[string]any{
			"kind":  map[string]any{"type": "string", "enum": []string{"account-review", "budget-review", "transaction-review", "credit-summary", "year-review"}},
			"month": map[string]any{"type": "string", "description": "YYYY-MM for budget and credit-card reviews; defaults to the current month."},
			"card":  map[string]any{"type": "string", "description": "Credit-card account name; empty reviews every card."},
			"year":  map[string]any{"type": "integer", "description": "Calendar year for year-review; defaults to the current year."},
		}), "kind"),
	})
	ai.Post("/health-commentary", guard.Wrap(h.HealthCommentary), RouteMeta{
		Summary:     "Structured commentary on the deterministic financial health score. " + reviewRecordSummary,
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
	ai.Post("/cashback-explain", guard.Wrap(h.ExplainCashback), RouteMeta{
		Summary: "Markdown walk-through of how a card period's cashback estimate was reached, payment by payment.",
		RequestBody: jsonBody(aiOptionProperties(map[string]any{
			"card":  schemaString,
			"month": map[string]any{"type": "string", "description": "YYYY-MM the period starts in; omit for the current period."},
		}), "card"),
	})
	ai.Post("/briefing", guard.Wrap(h.Briefing), RouteMeta{
		Summary:     "Dashboard intelligence briefing: markdown briefing, alerts and pattern cards, with the engine's current-month facts. " + reviewRecordSummary,
		RequestBody: optionalJSONBody(aiOptionProperties(nil)),
	})
	ai.Post("/budget-advisor", guard.Wrap(h.BudgetAdvice), RouteMeta{
		Summary:     "Budget advisor: suggested limits and alerts from the model; averages, velocity flags, 30-day projection and forecasted savings from the engine.",
		RequestBody: optionalJSONBody(aiOptionProperties(nil)),
	})
	ai.Post("/capture", guard.Wrap(h.Capture), RouteMeta{
		Summary: "Smart Capture: reads screenshots and pasted text into suggested transactions, streamed as NDJSON events: progress (stage, materialId, status, events, message), result (suggestions, unreadable, balances) and error (message). Writes nothing; materials are sent to the AI provider and never stored.",
		RequestBody: jsonBody(map[string]any{
			"materials": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
				"id":        schemaString,
				"kind":      map[string]any{"type": "string", "enum": []string{"image", "text"}},
				"name":      schemaString,
				"mediaType": map[string]any{"type": "string", "enum": []string{"image/png", "image/jpeg", "image/webp", "image/gif"}},
				"data":      map[string]any{"type": "string", "format": "byte", "description": "Base64 image bytes."},
				"text":      schemaString,
			}}},
			"model": schemaString,
		}, "materials"),
	})
	ai.Post("/chat", guard.Wrap(h.Chat), RouteMeta{
		Summary: "Tool-using chat streamed as NDJSON events: text (delta), tool (name, status), done (model) and error (message).",
		RequestBody: jsonBody(map[string]any{
			"messages": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"role": map[string]any{"type": "string", "enum": []string{"user", "assistant"}}, "content": schemaString}}},
			"page":     map[string]any{"type": "object", "properties": map[string]any{"path": schemaString, "title": schemaString, "summary": schemaString}},
			"model":    schemaString,
		}, "messages"),
	})
	ai.Post("/news-sentiment", guard.Wrap(h.NewsSentiment), RouteMeta{
		Summary:     "News scanner: web search on a topic, a per-article sentiment, confidence and impact from the model, and the bias by strict plurality (WM-INV-17).",
		RequestBody: jsonBody(aiOptionProperties(map[string]any{"topic": schemaString}), "topic"),
	})
	ai.Post("/ticker-insight", guard.Wrap(h.TickerInsight), RouteMeta{
		Summary: "Structured commentary on the engine's analysis of one symbol plus recent headlines (WM-AI-14).",
		RequestBody: jsonBody(aiOptionProperties(map[string]any{
			"symbol":    schemaString,
			"timeframe": map[string]any{"type": "string", "enum": []string{"1h", "4h", "1d", "1w"}},
		}), "symbol"),
	})
	ai.Post("/investment-analysis", guard.Wrap(h.InvestmentAnalysis), RouteMeta{
		Summary: "Investment think tank streamed as NDJSON events: phase (phase, status running|ok|fallback|error), context (the raw inputs and the focus: " +
			"source request|holdings and symbols), message (phase, name, content, model), actions (commands, and recorded: what the run wrote to the " +
			"recommendations list as source think-tank, or why not), done and error (message). Without symbols the run analyses the user's current holdings.",
		RequestBody: optionalJSONBody(map[string]any{
			"news": map[string]any{"type": "array", "description": "News scanner reports to use as pre-analysed context.", "items": map[string]any{"type": "object"}},
			"symbols": map[string]any{
				"type": "array", "maxItems": domain.ThinkTankFocusLimit, "items": schemaString,
				"description": "Symbols to analyse (e.g. FPT, BTC); empty means the user's current holdings.",
			},
			"model": schemaString,
		}),
	})
}
