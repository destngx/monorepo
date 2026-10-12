package ai

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
)

// marketTools let the chat search the web and read a ticker analysis; each
// exists only when its service is wired.
func (s *Service) marketTools(str func(string) map[string]any) []chatTool {
	var tools []chatTool
	if s.search != nil {
		tools = append(tools, chatTool{
			definition: domain.AITool{Name: "webSearch", Description: "Search the web for recent news, rates or facts. Cite the sources you use.", Parameters: objectSchema(map[string]any{"query": str("Search query")})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Query string `json:"query"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				return s.search.Search(ctx, input.Query)
			},
		})
	}
	if s.market != nil {
		tools = append(tools, chatTool{
			definition: domain.AITool{Name: "getMarketAnalysis", Description: "Technicals, multi-timeframe confluence, seasonality and risk for a VN stock, index, gold or FX symbol.", Parameters: objectSchema(map[string]any{
				"symbol":    str("Symbol such as FPT, VNINDEX, SJC or USDVND"),
				"timeframe": str("1h, 4h, 1d or 1w; default 1d"),
			})},
			run: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var input struct {
					Symbol    string `json:"symbol"`
					Timeframe string `json:"timeframe"`
				}
				if err := decodeArguments(arguments, &input); err != nil {
					return nil, err
				}
				timeframe, err := domain.ParseTimeframe(firstNonEmpty(input.Timeframe, string(domain.Timeframe1D)))
				if err != nil {
					return nil, err
				}
				analysis, err := s.market.Analysis(ctx, input.Symbol, timeframe)
				if err != nil {
					return nil, err
				}
				return tickerFacts(analysis), nil
			},
		})
	}
	return tools
}
