package mcp

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"strings"
)

// SetAIService exposes the AI use cases as tools (WM-MCP-4).
func (s *Server) SetAIService(aiService port.AIService) {
	s.aiService = aiService
}

var (
	modelSchema    = map[string]any{"type": "string", "description": "Model override for this call."}
	aiForceSchema  = map[string]any{"type": "boolean", "description": "Bypass the 30-minute result cache."}
	reviewKindEnum = []string{string(port.ReviewAccounts), string(port.ReviewTransactions), string(port.ReviewCreditCard), string(port.ReviewYear)}
)

type aiOptions struct {
	Force bool   `json:"force"`
	Model string `json:"model"`
}

func (s *Server) aiTools() []tool {
	ai := s.aiService
	optionsTool := func(name, description string, run func(ctx context.Context, options aiOptions) (any, error)) tool {
		return tool{
			definition: definition(name, description, object(map[string]any{"force": aiForceSchema, "model": modelSchema})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var options aiOptions
				if err := decodeArguments(arguments, &options); err != nil {
					return nil, err
				}
				return run(ctx, options)
			},
		}
	}
	return []tool{
		{
			definition: definition("GetAIStatus", "Reports AI configuration (endpoint host, models, fallback) and required prompts missing from the Prompts sheet.", nil),
			handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return ai.Status(ctx), nil
			},
		},
		{
			definition: definition("GetAIUsage", "Reports the OpenAI quota from the ai-gateway: rate-limit windows (used percent, reset time) or token totals for API-key accounts.", nil),
			handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				return ai.ProviderUsage(ctx)
			},
		},
		{
			definition: definition("GetAIModels", "Lists the model ids the AI endpoint offers.", nil),
			handler: func(ctx context.Context, _ json.RawMessage) (any, error) {
				models, err := ai.ProviderModels(ctx)
				return map[string]any{"models": models}, err
			},
		},
		{
			definition: definition("SuggestCategory", "Suggests a category for a payee: the payee's history first, then the model, limited to known categories.", object(map[string]any{"payee": stringSchema, "model": modelSchema}, "payee")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Payee string `json:"payee"`
					Model string `json:"model"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return ai.SuggestCategory(ctx, args.Payee, args.Model)
			},
		},
		{
			definition: definition("ReviewFinances", "Structured AI review of accounts, budget, the last 7 days of transactions, credit cards, or a calendar year of income and expense.", object(map[string]any{
				"kind":  map[string]any{"type": "string", "enum": reviewKindEnum},
				"month": map[string]any{"type": "string", "description": "YYYY-MM for budget and credit-card reviews."},
				"card":  map[string]any{"type": "string", "description": "Credit-card account name; empty reviews every card."},
				"year":  map[string]any{"type": "integer", "description": "Calendar year for year-review; defaults to the current year."},
				"force": aiForceSchema,
				"model": modelSchema,
			}, "kind")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.ReviewRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				return ai.Review(ctx, request)
			},
		},
		optionsTool("GetHealthCommentary", "Structured AI commentary on the deterministic financial health score.", func(ctx context.Context, options aiOptions) (any, error) {
			return ai.HealthCommentary(ctx, options.Force, options.Model)
		}),
		optionsTool("GetIntelligenceBriefing", "AI dashboard briefing with alerts and pattern cards, plus the engine's current-month facts.", func(ctx context.Context, options aiOptions) (any, error) {
			return ai.Briefing(ctx, options.Force, options.Model)
		}),
		optionsTool("GetBudgetAdvice", "AI budget advisor: suggested limits and alerts, with engine averages, velocity flags, 30-day projection and forecasted savings.", func(ctx context.Context, options aiOptions) (any, error) {
			return ai.BudgetAdvice(ctx, options.Force, options.Model)
		}),
		{
			definition: definition("GetChartInsight", "Markdown AI commentary on chart data (at most 12 KB).", object(map[string]any{
				"chartType": stringSchema, "title": stringSchema, "description": stringSchema, "timeframe": stringSchema,
				"data": map[string]any{"description": "The chart series."}, "force": aiForceSchema, "model": modelSchema,
			}, "chartType", "data")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.ChartInsightRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				insight, err := ai.ChartInsight(ctx, request)
				if err != nil {
					return nil, err
				}
				return map[string]any{"insight": insight}, nil
			},
		},
		{
			definition: definition("ExplainCashback", "Markdown walk-through of how a card period's cashback estimate was reached: each payment's rule, minimums and caps.", object(map[string]any{
				"card": stringSchema, "month": map[string]any{"type": "string", "description": "YYYY-MM the period starts in; omit for the current period."},
				"force": aiForceSchema, "model": modelSchema,
			}, "card")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.CashbackExplainRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				explanation, err := ai.ExplainCashback(ctx, request)
				if err != nil {
					return nil, err
				}
				return map[string]any{"explanation": explanation}, nil
			},
		},
		{
			definition: definition("SuggestTransactionsFromMaterials", "Smart Capture: reads bank or wallet screenshots and pasted texts (notifications, SMS, statements) into suggested transactions with matched accounts, categories, transfer pairs, ledger duplicates and reported balances. Writes nothing; record accepted rows with CreateTransaction.", object(map[string]any{
				"materials": map[string]any{"type": "array", "minItems": 1, "items": object(map[string]any{
					"id":        map[string]any{"type": "string", "description": "Unique id within the batch, used as evidence."},
					"kind":      map[string]any{"type": "string", "enum": []string{string(domain.CaptureImage), string(domain.CaptureText)}},
					"name":      map[string]any{"type": "string", "description": "Optional file name."},
					"mediaType": map[string]any{"type": "string", "enum": domain.CaptureImageTypes, "description": "Required for images."},
					"data":      map[string]any{"type": "string", "contentEncoding": "base64", "description": "Base64 image bytes for kind image."},
					"text":      map[string]any{"type": "string", "description": "The text for kind text."},
				}, "id", "kind")},
				"model": modelSchema,
			}, "materials")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.CaptureRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				return ai.Capture(ctx, request, nil)
			},
		},
		{
			definition: definition("Chat", "Asks the tool-using finance assistant; returns the full reply and the tools it used.", object(map[string]any{
				"messages": map[string]any{"type": "array", "items": object(map[string]any{"role": map[string]any{"type": "string", "enum": []string{"user", "assistant"}}, "content": stringSchema}, "role", "content")},
				"model":    modelSchema,
			}, "messages")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.ChatRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				var reply strings.Builder
				toolsUsed := []string{}
				model := ""
				err := ai.Chat(ctx, request, func(event port.ChatEvent) error {
					switch event.Type {
					case "text":
						reply.WriteString(event.Delta)
					case "tool":
						if event.Status == "running" {
							toolsUsed = append(toolsUsed, event.Tool)
						}
					case "done":
						model = event.Model
					}
					return nil
				})
				if err != nil {
					return nil, err
				}
				return map[string]any{"reply": reply.String(), "tools": toolsUsed, "model": model}, nil
			},
		},
	}
}
