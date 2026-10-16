package mcp

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"encoding/json"
)

// SetMarketAnalysisService exposes the market pulse and ticker deep-dive
// (WM-MCP-4).
func (s *Server) SetMarketAnalysisService(analysis port.MarketAnalysisService) {
	s.marketAnalysis = analysis
}

// SetPortfolioService exposes the investments valuation (WM-MCP-4).
func (s *Server) SetPortfolioService(portfolio port.PortfolioService) {
	s.portfolio = portfolio
}

var (
	timeframeSchema   = map[string]any{"type": "string", "enum": timeframeNames(), "default": string(domain.Timeframe1D)}
	marketForceSchema = map[string]any{"type": "boolean", "description": "Bypass the market cache and fetch fresh provider data."}
)

func timeframeNames() []string {
	names := make([]string, 0, len(domain.Timeframes))
	for _, timeframe := range domain.Timeframes {
		names = append(names, string(timeframe))
	}
	return names
}

// parseTimeframe defaults an empty timeframe to 1d, as the HTTP routes do.
func parseTimeframe(raw string) (domain.Timeframe, error) {
	if raw == "" {
		return domain.Timeframe1D, nil
	}
	return domain.ParseTimeframe(raw)
}

func (s *Server) marketTools() []tool {
	market := s.marketService
	type marketArgs struct {
		Provider string `json:"provider"`
		Symbol   string `json:"symbol"`
		Type     string `json:"type"`
		From     string `json:"from"`
		To       string `json:"to"`
	}
	marketTool := func(name, description string, schema map[string]any, call func(context.Context, marketArgs) (any, error)) tool {
		return tool{
			definition: definition(name, description, schema),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args marketArgs
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				return call(ctx, args)
			},
		}
	}
	tickerTypes := []string{string(domain.TickerTypeEquity), string(domain.TickerTypeIFC), string(domain.TickerTypeGold), string(domain.TickerTypeCrypto)}
	seriesTypes := []string{string(domain.SeriesTypeEquity), string(domain.SeriesTypeGoldUSD), string(domain.SeriesTypeForex)}
	return []tool{
		marketTool("GetMarketProviderHealth", "Reports a market data provider's health (fmarket, vnstock, coingecko, currencyapi or yahoo).",
			object(map[string]any{"provider": map[string]any{"type": "string", "enum": []string{"fmarket", "vnstock", "coingecko", "currencyapi", "yahoo"}}}, "provider"),
			func(ctx context.Context, args marketArgs) (any, error) { return market.Health(ctx, args.Provider) }),
		marketTool("GetMarketTicker", "Returns the latest quote: equity via vnstock, fund certificate (ifc) via fmarket, gold, or crypto in USD.",
			object(map[string]any{"symbol": stringSchema, "type": map[string]any{"type": "string", "enum": tickerTypes}}, "symbol", "type"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetTicker(ctx, args.Symbol, domain.TickerType(args.Type))
			}),
		marketTool("GetExchangeRate", "Returns an exchange rate, e.g. USD to VND.",
			object(map[string]any{"from": stringSchema, "to": stringSchema}, "from", "to"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetExchangeRate(ctx, args.From, args.To)
			}),
		marketTool("GetPriceSeries", "Returns a historical price series for an equity, gold in USD, or a forex pair.",
			object(map[string]any{"symbol": stringSchema, "type": map[string]any{"type": "string", "enum": seriesTypes}}, "type"),
			func(ctx context.Context, args marketArgs) (any, error) {
				return market.GetPriceSeries(ctx, args.Symbol, domain.SeriesType(args.Type))
			}),
		marketTool("GetBankRates", "Returns current bank deposit interest rates.", nil,
			func(ctx context.Context, _ marketArgs) (any, error) { return market.GetBankInterestRate(ctx) }),
		{
			definition: definition("GetMarketCandles", "OHLCV bars for a VN equity, index, FX pair, gold or VIX, with the provider that served them and any fallback reason.", object(map[string]any{
				"symbol":    map[string]any{"type": "string", "description": "Ticker symbol, e.g. FPT or VNINDEX."},
				"timeframe": timeframeSchema,
				"days":      map[string]any{"type": "integer", "minimum": 1, "description": "Lookback in days; defaults to enough history for EMA200."},
				"force":     marketForceSchema,
			}, "symbol")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Symbol    string `json:"symbol"`
					Timeframe string `json:"timeframe"`
					Days      int    `json:"days"`
					Force     bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				timeframe, err := parseTimeframe(args.Timeframe)
				if err != nil {
					return nil, err
				}
				return market.GetCandles(domain.WithBypassCache(ctx, args.Force), args.Symbol, timeframe, args.Days)
			},
		},
	}
}

func (s *Server) marketAnalysisTools() []tool {
	analysis := s.marketAnalysis
	return []tool{
		{
			definition: definition("GetMarketPulse", "VN market pulse: per-instrument change, direction and momentum, top movers, correlations, capital flow, scenario, VN-Index technicals and sentiment (WM-RULE-9).", object(map[string]any{
				"timeframe": timeframeSchema, "force": marketForceSchema,
			})),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Timeframe string `json:"timeframe"`
					Force     bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				timeframe, err := parseTimeframe(args.Timeframe)
				if err != nil {
					return nil, err
				}
				return analysis.Pulse(domain.WithBypassCache(ctx, args.Force), timeframe)
			},
		},
		{
			definition: definition("GetMarketAnalysis", "Ticker deep-dive: technicals, 1h/4h/1d/1w confluence, seasonality and history-only risk statistics. Signals are rule-based heuristics.", object(map[string]any{
				"symbol":    map[string]any{"type": "string", "description": "Ticker or instrument, e.g. FPT, VNINDEX, SJC, USDVND."},
				"timeframe": timeframeSchema,
				"force":     marketForceSchema,
			}, "symbol")),
			handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
				var args struct {
					Symbol    string `json:"symbol"`
					Timeframe string `json:"timeframe"`
					Force     bool   `json:"force"`
				}
				if err := decodeArguments(arguments, &args); err != nil {
					return nil, err
				}
				timeframe, err := parseTimeframe(args.Timeframe)
				if err != nil {
					return nil, err
				}
				return analysis.Analysis(domain.WithBypassCache(ctx, args.Force), args.Symbol, timeframe)
			},
		},
	}
}

func (s *Server) portfolioTools() []tool {
	portfolio := s.portfolio
	return []tool{{
		definition: definition("GetInvestmentValuation", "Values crypto and fund holdings in VND with the USD/USDT rates used, per-holding issues and totals (WM-RULE-8).", object(map[string]any{"force": forceSchema})),
		handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
			var args struct {
				Force bool `json:"force"`
			}
			if err := decodeArguments(arguments, &args); err != nil {
				return nil, err
			}
			return portfolio.Valuation(ctx, args.Force)
		},
	}, {
		definition: definition("GetAssetAllocation", "The one asset allocation (cash, fund certificates, crypto, and money lent without a target) with values, shares, targets from the Preferences tab and drift; rebalance when a class is more than 5 points off target.", object(map[string]any{"force": forceSchema})),
		handler: func(ctx context.Context, arguments json.RawMessage) (any, error) {
			var args struct {
				Force bool `json:"force"`
			}
			if err := decodeArguments(arguments, &args); err != nil {
				return nil, err
			}
			return portfolio.Allocation(ctx, args.Force)
		},
	}}
}
