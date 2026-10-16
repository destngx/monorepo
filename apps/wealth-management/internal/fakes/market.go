package fakes

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"sync"
)

// MarketCall records one market call: the symbol, the timeframe or type,
// the lookback and whether the context asked to bypass the cache.
type MarketCall struct {
	Symbol    string
	Kind      string
	Days      int
	Bypass    bool
	ForceRead bool
}

// Market is a port.MarketProviderService, MarketAnalysisService and
// PortfolioService with canned answers. Err fails every call.
type Market struct {
	mu    sync.Mutex
	Err   error
	Calls []MarketCall
}

var (
	_ port.MarketProviderService = (*Market)(nil)
	_ port.MarketAnalysisService = (*Market)(nil)
	_ port.PortfolioService      = (*Market)(nil)
	_ port.FmarketService        = (*Market)(nil)
)

func (f *Market) record(ctx context.Context, call MarketCall) {
	f.mu.Lock()
	defer f.mu.Unlock()
	call.Bypass = domain.ShouldBypassCache(ctx)
	f.Calls = append(f.Calls, call)
}

func (f *Market) Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error) {
	f.record(ctx, MarketCall{Kind: "health:" + provider})
	return domain.MarketProviderHealth{Provider: provider, Status: "ok"}, f.Err
}

func (f *Market) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	f.record(ctx, MarketCall{Symbol: symbol, Kind: string(tickerType)})
	return domain.Ticker{Symbol: symbol, Type: tickerType, Provider: "fake"}, f.Err
}

func (f *Market) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	f.record(ctx, MarketCall{Symbol: from + "_" + to})
	return domain.ExchangeRate{From: from, To: to, Rate: 25000, Provider: "fake"}, f.Err
}

func (f *Market) GetPriceSeries(ctx context.Context, symbol string, seriesType domain.SeriesType) (domain.PriceSeries, error) {
	f.record(ctx, MarketCall{Symbol: symbol, Kind: string(seriesType)})
	return domain.PriceSeries{Provider: "fake", SeriesType: seriesType}, f.Err
}

func (f *Market) GetBankInterestRate(ctx context.Context) (domain.BankRates, error) {
	f.record(ctx, MarketCall{Kind: "bank-rates"})
	return domain.BankRates{Rates: []domain.BankRate{{Bank: "Test Bank", Rate: 6.5}}, Source: &domain.DataSource{Provider: "fake"}}, f.Err
}

func (f *Market) GetCandles(ctx context.Context, symbol string, timeframe domain.Timeframe, lookbackDays int) (domain.CandleSeries, error) {
	f.record(ctx, MarketCall{Symbol: symbol, Kind: string(timeframe), Days: lookbackDays})
	return domain.CandleSeries{Symbol: symbol, Timeframe: timeframe, Provider: "fake"}, f.Err
}

func (f *Market) Pulse(ctx context.Context, timeframe domain.Timeframe) (domain.MarketPulse, error) {
	f.record(ctx, MarketCall{Kind: "pulse:" + string(timeframe)})
	return domain.MarketPulse{}, f.Err
}

func (f *Market) Analysis(ctx context.Context, symbol string, timeframe domain.Timeframe) (domain.MarketAnalysis, error) {
	f.record(ctx, MarketCall{Symbol: symbol, Kind: "analysis:" + string(timeframe)})
	return domain.MarketAnalysis{}, f.Err
}

func (f *Market) Valuation(ctx context.Context, forceFresh bool) (domain.PortfolioValuation, error) {
	f.record(ctx, MarketCall{Kind: "valuation", ForceRead: forceFresh})
	return domain.PortfolioValuation{}, f.Err
}

func (f *Market) Allocation(ctx context.Context, forceFresh bool) (domain.Allocation, error) {
	f.record(ctx, MarketCall{Kind: "allocation", ForceRead: forceFresh})
	return domain.ComputeAllocation(domain.AllocationInputs{}), f.Err
}

func (f *Market) FundPerformance(ctx context.Context, forceFresh bool) (domain.FundPerformance, error) {
	f.record(ctx, MarketCall{Kind: "fund-performance", ForceRead: forceFresh})
	return domain.ComputeFundPerformance(domain.FundPerformanceInputs{}), f.Err
}

func (f *Market) RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error) {
	f.record(ctx, MarketCall{Kind: "fmarket:" + string(request.Action), ForceRead: forceFresh})
	return map[string]any{"data": map[string]any{}}, f.Err
}
