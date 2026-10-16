package mcp

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
	"errors"
)

// errInvestmentAnalysis reports an error event of the think-tank stream.
var errInvestmentAnalysis = errors.New("investment analysis failed")

// investmentAnalysisResult is the think-tank stream collected into one
// answer, since MCP tool calls return a single result.
type investmentAnalysisResult struct {
	Phases   []port.InvestmentEvent     `json:"phases"`
	Messages []port.InvestmentEvent     `json:"messages"`
	Commands []domain.InvestmentCommand `json:"commands"`
	Context  any                        `json:"context,omitempty"`
	// Recorded is what the run wrote to the recommendations list (WM-REC-8).
	Recorded *domain.RecommendationRecord `json:"recorded,omitempty"`
}

// thinkTankSymbolsSchema is the optional think tank focus (WM-INV-29).
var thinkTankSymbolsSchema = map[string]any{
	"type": "array", "maxItems": domain.ThinkTankFocusLimit, "items": map[string]any{"type": "string"},
	"description": "Symbols to analyse (e.g. FPT, BTC); empty means the user's current holdings.",
}

func (s *Server) aiInvestmentTools() []tool {
	ai := s.aiService
	return []tool{
		{
			definition: definition("GetNewsSentiment", "News scanner: web search on a topic, per-article sentiment, confidence and impact, and the overall bias (WM-INV-17).", object(map[string]any{
				"topic": stringSchema, "force": aiForceSchema, "model": modelSchema,
			}, "topic")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.NewsRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				return ai.NewsSentiment(ctx, request)
			},
		},
		{
			definition: definition("GetTickerInsight", "Structured AI commentary on the engine's analysis of one symbol plus recent headlines (WM-AI-14).", object(map[string]any{
				"symbol": stringSchema, "timeframe": timeframeSchema, "force": aiForceSchema, "model": modelSchema,
			}, "symbol")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.TickerInsightRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				return ai.TickerInsight(ctx, request)
			},
		},
		{
			definition: definition("RunInvestmentAnalysis", "Investment think tank: context (with the focus), analyst messages, synthesis and suggested commands, collected from the stream into one result. Without symbols it analyses the user's current holdings. Its recommendations (0-3) and outcomes for its earlier items go to the recommendations list (source think-tank); recorded says what was written, or why not.", object(map[string]any{
				"news":    map[string]any{"type": "array", "description": "News scanner reports to use as pre-analysed context.", "items": map[string]any{"type": "object"}},
				"symbols": thinkTankSymbolsSchema,
				"model":   modelSchema,
			})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var request port.InvestmentAnalysisRequest
				if err := decodeArguments(arguments, &request); err != nil {
					return nil, err
				}
				return collectInvestmentAnalysis(ctx, ai, request)
			},
		},
	}
}

func collectInvestmentAnalysis(ctx context.Context, ai port.AIService, request port.InvestmentAnalysisRequest) (investmentAnalysisResult, error) {
	result := investmentAnalysisResult{Phases: []port.InvestmentEvent{}, Messages: []port.InvestmentEvent{}, Commands: []domain.InvestmentCommand{}}
	var streamErr error
	err := ai.InvestmentAnalysis(ctx, request, func(event port.InvestmentEvent) error {
		switch event.Type {
		case "phase":
			result.Phases = append(result.Phases, event)
		case "message":
			result.Messages = append(result.Messages, event)
		case "actions":
			result.Commands = append(result.Commands, event.Commands...)
			result.Recorded = event.Recorded
		case "context":
			result.Context = event.Context
		case "error":
			streamErr = errors.Join(streamErr, errors.New(event.Message))
		}
		return nil
	})
	if err != nil {
		return investmentAnalysisResult{}, err
	}
	if streamErr != nil {
		return investmentAnalysisResult{}, errors.Join(errInvestmentAnalysis, streamErr)
	}
	return result, nil
}
