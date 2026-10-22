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
	return domain.BankRates{Rates: fakeBankRates(), Source: &domain.DataSource{Provider: "fake"}}, f.Err
}

func (f *Market) GetFlexibleSavings(ctx context.Context) (domain.FlexibleSavings, error) {
	f.record(ctx, MarketCall{Kind: "flexible-savings"})
	return domain.FlexibleSavings{Products: fakeFlexibleSavings(), Failed: []string{}, Source: &domain.DataSource{Provider: "fake"}}, f.Err
}

// fakeFlexibleSavings is one invented wallet product paying from the next
// working day, below the fake 1-month rate.
func fakeFlexibleSavings() []domain.FlexibleSaving {
	return []domain.FlexibleSaving{{
		ID: "fake:wallet", Provider: "Test Wallet", Product: "Earn", Rate: 4, Backing: domain.SavingsBackingDeposit,
		EarningStarts: domain.EarningStartsNextWorkingDay, Currency: "VND", URL: "https://wallet.example/earn",
	}}
}

// fakeBankRates is the market server's tenored shape: a no-term, a
// 1-month and a 12-month rate of one invented bank.
func fakeBankRates() []domain.BankRate {
	noTerm, oneMonth, twelve := 0, 1, 12
	return []domain.BankRate{
		{Bank: "Test Bank", Tenor: "no-term", TenorMonths: &noTerm, InstantAccess: true, Channel: "counter", Rate: 0.5, Currency: "VND"},
		{Bank: "Test Bank", Tenor: "1m", TenorMonths: &oneMonth, Channel: "counter", Rate: 4.5, Currency: "VND"},
		{Bank: "Test Bank", Tenor: "12m", TenorMonths: &twelve, Channel: "counter", Rate: 6.5, Currency: "VND"},
	}
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

func (f *Market) Exposure(ctx context.Context, forceFresh bool) (domain.Exposure, error) {
	f.record(ctx, MarketCall{Kind: "exposure", ForceRead: forceFresh})
	return domain.ComputeExposure(domain.ExposureInputs{}), f.Err
}

func (f *Market) Hedges(ctx context.Context, forceFresh bool) (domain.Hedges, error) {
	f.record(ctx, MarketCall{Kind: "hedges", ForceRead: forceFresh})
	return domain.ComputeHedges(domain.HedgeInputs{}), f.Err
}

func (f *Market) RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error) {
	f.record(ctx, MarketCall{Kind: "fmarket:" + string(request.Action), ForceRead: forceFresh})
	return map[string]any{"data": map[string]any{}}, f.Err
}
