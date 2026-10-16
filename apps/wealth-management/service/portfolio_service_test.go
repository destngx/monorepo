package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// portfolioDB serves holdings and accounts; other methods are unused.
type portfolioDB struct {
	port.DatabaseService
	assets      domain.InvestmentAssets
	accounts    []domain.Account
	err         error
	forceFreshs []bool
}

func (d *portfolioDB) GetInvestmentAssets(forceFresh bool) (domain.InvestmentAssets, error) {
	d.forceFreshs = append(d.forceFreshs, forceFresh)
	return d.assets, d.err
}

func (d *portfolioDB) ListAccounts(bool) ([]domain.Account, error) {
	return d.accounts, nil
}

// portfolioMarket answers from fixed maps and records lookups.
type portfolioMarket struct {
	port.MarketProviderService
	mu          sync.Mutex
	tickers     map[string]domain.Ticker
	rates       map[string]domain.ExchangeRate
	lookups     []string
	bypassCache bool
}

func (m *portfolioMarket) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lookups = append(m.lookups, string(tickerType)+":"+symbol)
	m.bypassCache = domain.ShouldBypassCache(ctx)
	ticker, ok := m.tickers[symbol]
	if !ok {
		return domain.Ticker{}, errors.New("unknown symbol")
	}
	return ticker, nil
}

func (m *portfolioMarket) GetExchangeRate(_ context.Context, from string, to string) (domain.ExchangeRate, error) {
	rate, ok := m.rates[from+"_"+to]
	if !ok {
		return domain.ExchangeRate{}, errors.New("all providers failed")
	}
	return rate, nil
}

func TestGivenHoldingsWhenValuationThenPricesEachSymbolOnceAndReportsMarketIssues(t *testing.T) {
	quantity := 2.0
	db := &portfolioDB{
		assets: domain.InvestmentAssets{
			Crypto: domain.InvestmentHoldingSection{Items: []domain.Holding{
				{Platform: "Binance", Symbol: "eth", Quantity: &quantity, Currency: domain.CurrencyUSD},
				{Platform: "Ledger", Symbol: "ETH", Quantity: &quantity, Currency: domain.CurrencyUSD},
				{Platform: "Ledger", Symbol: "ZZZ", Quantity: &quantity, Currency: domain.CurrencyUSD},
			}},
			Funds: domain.InvestmentHoldingSection{Items: []domain.Holding{
				{Platform: "Fmarket", Symbol: "VESAF", Quantity: &quantity, Currency: domain.CurrencyVND},
			}},
		},
		accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Currency: domain.CurrencyVND, Balance: 1000}},
	}
	market := &portfolioMarket{
		tickers: map[string]domain.Ticker{
			"ETH": {Symbol: "ETH", Price: 3000, Currency: "USD", Provider: "coingecko"},
			"VESAF": {Symbol: "VESAF", Price: 31000, Currency: "VND", Provider: "fmarket", Timestamp: "2026-10-12T00:00:00Z",
				Source: &domain.DataSource{Provider: "fmarket", Stale: true, AsOf: "2026-10-11T00:00:00Z"}},
		},
		rates: map[string]domain.ExchangeRate{"USD_VND": {From: "USD", To: "VND", Rate: 26000, Provider: "yahoo"}},
	}
	now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	service := NewPortfolioService(db, market, func() time.Time { return now })

	result, err := service.Valuation(context.Background(), true)
	if err != nil {
		t.Fatalf("valuation error: %v", err)
	}
	if len(market.lookups) != 3 || !market.bypassCache || !db.forceFreshs[0] {
		t.Fatalf("expected 3 fresh lookups, got %v (bypass %v)", market.lookups, market.bypassCache)
	}
	if *result.Holdings[1].ValueVND != 2*3000*26000 || *result.Holdings[3].ValueVND != 62000 {
		t.Fatalf("unexpected values: %#v", result.Holdings)
	}
	if source := result.Holdings[3].LivePrice.Source; source == nil || !source.Stale {
		t.Fatalf("expected the stale ticker source on the live price, got %#v", result.Holdings[3].LivePrice)
	}
	if result.Holdings[2].Issue != "no live price: unknown symbol" {
		t.Fatalf("expected the lookup error on ZZZ, got %q", result.Holdings[2].Issue)
	}
	if len(result.Issues) != 1 || result.Issues[0] != "USDT/VND rate unavailable: all providers failed" {
		t.Fatalf("expected the USDT rate issue, got %v", result.Issues)
	}
	if result.Totals.Total != 1000+62000+2*2*3000*26000 || !result.GeneratedAt.Equal(now) {
		t.Fatalf("unexpected totals: %#v", result.Totals)
	}
}

func TestGivenSheetFailureWhenValuationThenFails(t *testing.T) {
	service := NewPortfolioService(&portfolioDB{err: domain.ErrNotFound}, &portfolioMarket{}, nil)
	if _, err := service.Valuation(context.Background(), false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected the sheet error, got %v", err)
	}
}

// allocationDB adds the ledger and the Preferences tab to portfolioDB.
type allocationDB struct {
	portfolioDB
	transactions []domain.Transaction
	prefs        domain.Preferences
	prefsErr     error
}

func (d *allocationDB) ReadLedger(bool) (domain.LedgerSnapshot, error) {
	return domain.LedgerSnapshot{Accounts: d.accounts, Transactions: d.transactions}, nil
}

func (d *allocationDB) ListPreferences(bool) (domain.Preferences, error) {
	return d.prefs, d.prefsErr
}

func TestGivenHoldingsAccountsMoneyLentAndTargetsWhenAllocatingThenOneAllocationCombinesThem(t *testing.T) {
	value := 40_000_000.0
	lent := domain.Transaction{Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Payee: "Person A", Category: "Lending / Repayment", CategoryNature: domain.NatureLending, Payment: ptr(10_000_000.0)}
	db := &allocationDB{
		portfolioDB: portfolioDB{
			assets:   domain.InvestmentAssets{Funds: domain.InvestmentHoldingSection{Items: []domain.Holding{{Platform: "Fmarket", Symbol: "FUND", Value: &value, Currency: domain.CurrencyVND}}}},
			accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 60_000_000}},
		},
		transactions: []domain.Transaction{lent},
		prefs:        domain.ParsePreferences([][]string{{"allocation.target.cash", "50"}, {"allocation.target.funds", "50"}, {"allocation.target.crypto", "0"}, {"bad.key", "1"}}, 2),
	}
	allocation, err := NewPortfolioService(db, nil, nil).Allocation(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if allocation.Line(domain.AllocationFunds).Value != 40_000_000 || allocation.Line(domain.AllocationCash).Value != 60_000_000 || allocation.Line(domain.AllocationMoneyLent).Value != 10_000_000 {
		t.Fatalf("lines %+v", allocation.Lines)
	}
	if allocation.Targets == nil || !allocation.Rebalance || *allocation.Line(domain.AllocationCash).Drift != 10 {
		t.Fatalf("targets and drift %+v", allocation)
	}
	// without a market server the rates are issues; the bad preference row is reported too
	if len(allocation.Issues) != 3 {
		t.Fatalf("issues %q", allocation.Issues)
	}
}

func TestGivenNoMarketWhenValuingThenSheetValuesCountAndLivePricesAreIssues(t *testing.T) {
	quantity := 1.0
	db := &portfolioDB{assets: domain.InvestmentAssets{Crypto: domain.InvestmentHoldingSection{Items: []domain.Holding{{Platform: "Wallet", Symbol: "BTC", Quantity: &quantity}}}}}
	valuation, err := NewPortfolioService(db, nil, nil).Valuation(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if valuation.Totals.Unvalued != 1 || valuation.Holdings[0].Issue == "" {
		t.Fatalf("valuation %+v", valuation)
	}
}

func TestGivenAPreferencesFailureWhenAllocatingThenItFails(t *testing.T) {
	db := &allocationDB{prefsErr: domain.ErrConflict}
	if _, err := NewPortfolioService(db, nil, nil).Allocation(context.Background(), false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v", err)
	}
}

// fundNAVs serves NAV histories by fund code and records the lookups.
type fundNAVs struct {
	mu      sync.Mutex
	points  map[string][]domain.NavPoint
	lookups []string
	bypass  bool
}

func (f *fundNAVs) GetFundNAVHistory(ctx context.Context, code string) (domain.FundNAVHistory, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookups = append(f.lookups, code)
	f.bypass = domain.ShouldBypassCache(ctx)
	points, ok := f.points[code]
	if !ok {
		return domain.FundNAVHistory{}, errors.New("fund not found")
	}
	return domain.FundNAVHistory{Symbol: code, Points: points}, nil
}

type bankRatesMarket struct {
	port.MarketProviderService
	rates []domain.BankRate
	err   error
}

func (m *bankRatesMarket) GetBankInterestRate(context.Context) (domain.BankRates, error) {
	return domain.BankRates{Rates: m.rates}, m.err
}

func TestGivenFundHoldingsWhenComputingFundPerformanceThenEachCodeIsLookedUpOnceAndComparedWithTheBestRate(t *testing.T) {
	ten, two := 10.0, 2.0
	db := &portfolioDB{assets: domain.InvestmentAssets{Funds: domain.InvestmentHoldingSection{Items: []domain.Holding{
		{Platform: "Platform A", Symbol: "FUND-A", Quantity: &ten, History: []domain.FundTrade{{Date: "2025-01-01", Units: 10}}},
		{Platform: "Platform B", Symbol: "fund-a", Quantity: &two},
		{Platform: "Platform B", Symbol: "FUND-X", Quantity: &two},
	}}}}
	navs := &fundNAVs{points: map[string][]domain.NavPoint{"FUND-A": {{Date: "2025-01-01", NAV: 10_000}, {Date: "2026-01-01", NAV: 11_000}}}}
	market := &bankRatesMarket{rates: []domain.BankRate{{Bank: "Bank A", Rate: 4}, {Bank: "Bank B", Rate: 5}}}
	now := func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }

	result, err := NewPortfolioService(db, market, now, WithFundNAVs(navs)).FundPerformance(context.Background(), true)

	if err != nil {
		t.Fatal(err)
	}
	if len(navs.lookups) != 2 || !navs.bypass || !db.forceFreshs[0] {
		t.Fatalf("lookups = %v, bypass = %v", navs.lookups, navs.bypass)
	}
	if result.Deposit == nil || result.Deposit.Bank != "Bank B" || result.Holdings[0].Return == nil || result.Holdings[0].Return.VsDeposit != domain.VsDepositAhead {
		t.Fatalf("result = %+v", result)
	}
	if result.Holdings[2].CostStatus != domain.FundCostNAVUnavailable || result.Totals.Value != 132_000 {
		t.Fatalf("holdings = %+v, totals = %+v", result.Holdings, result.Totals)
	}
}

func TestGivenNoMarketWhenComputingFundPerformanceThenValuesAndCostAreUnknownWithTheReason(t *testing.T) {
	one := 1.0
	db := &portfolioDB{assets: domain.InvestmentAssets{Funds: domain.InvestmentHoldingSection{Items: []domain.Holding{{Symbol: "FUND-A", Quantity: &one}}}}}

	result, err := NewPortfolioService(db, nil, nil).FundPerformance(context.Background(), false)

	if err != nil || result.Deposit != nil || result.DepositIssue == "" || result.Holdings[0].CostStatus != domain.FundCostNAVUnavailable {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}

func TestGivenTheSheetFailsWhenComputingFundPerformanceThenTheCallFails(t *testing.T) {
	db := &portfolioDB{err: domain.ErrNotFound}
	if _, err := NewPortfolioService(db, nil, nil).FundPerformance(context.Background(), false); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

// seriesMarket adds candles by symbol to portfolioMarket.
type seriesMarket struct {
	portfolioMarket
	candles  map[string]domain.CandleSeries
	requests []string
}

func (m *seriesMarket) GetCandles(ctx context.Context, symbol string, timeframe domain.Timeframe, days int) (domain.CandleSeries, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests = append(m.requests, fmt.Sprintf("%s %s %d", symbol, timeframe, days))
	m.bypassCache = domain.ShouldBypassCache(ctx)
	series, ok := m.candles[symbol]
	if !ok {
		return domain.CandleSeries{}, errors.New("no bars")
	}
	return series, nil
}

func hourly(start time.Time, closes ...float64) domain.CandleSeries {
	series := domain.CandleSeries{Provider: "yahoo"}
	for i, close := range closes {
		series.Candles = append(series.Candles, domain.Candle{Time: start.Add(time.Duration(i) * time.Hour), Close: close})
	}
	return series
}

func exposureDB() *allocationDB {
	one, ten := 1.0, 10.0
	return &allocationDB{portfolioDB: portfolioDB{
		assets: domain.InvestmentAssets{
			Crypto: domain.InvestmentHoldingSection{Items: []domain.Holding{
				{Platform: "Exchange A", Symbol: "btc", Quantity: &one, Currency: domain.CurrencyUSD},
				{Platform: "Wallet B", Symbol: "BTC", Quantity: &one, Currency: domain.CurrencyUSD},
				{Platform: "Wallet B", Symbol: "ZZZ", Quantity: &one, Currency: domain.CurrencyUSD},
			}},
			Funds: domain.InvestmentHoldingSection{Items: []domain.Holding{{Platform: "Fund platform", Symbol: "FUND-A", Quantity: &ten, Currency: domain.CurrencyVND}}},
		},
		accounts: []domain.Account{{Name: "Bank", Type: domain.AccountTypeBank, Balance: 100_000_000}},
	}}
}

func TestGivenHoldingsWhenComputingExposureThenEachCoinAndFundCodeIsMovedOnce(t *testing.T) {
	start := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = 100
	}
	closes[24] = 110
	market := &seriesMarket{
		portfolioMarket: portfolioMarket{
			tickers: map[string]domain.Ticker{
				"BTC":    {Symbol: "BTC", Price: 1000, Currency: "USD"},
				"FUND-A": {Symbol: "FUND-A", Price: 20_000, Currency: "VND"},
			},
			rates: map[string]domain.ExchangeRate{"USD_VND": {From: "USD", To: "VND", Rate: 25_000}},
		},
		candles: map[string]domain.CandleSeries{"BTC-USD": hourly(start, closes...)},
	}
	navs := &fundNAVs{points: map[string][]domain.NavPoint{"FUND-A": {{Date: "2026-10-15", NAV: 20_000}, {Date: "2026-10-16", NAV: 20_200}}}}
	now := func() time.Time { return start.Add(25 * time.Hour) }

	exposure, err := NewPortfolioService(exposureDB(), market, now, WithFundNAVs(navs)).Exposure(context.Background(), true)

	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(market.requests)
	if strings.Join(market.requests, ",") != "BTC-USD 1h 2,ZZZ-USD 1h 2" || !market.bypassCache || len(navs.lookups) != 1 || !navs.bypass {
		t.Fatalf("requests %v navs %v", market.requests, navs.lookups)
	}
	// BTC 2 × 1000 USD × 25,000 = 50M after +10%; fund 200k after +1%
	crypto := exposure.Crypto
	if len(crypto.Coins) != 2 || crypto.Coins[0].Symbol != "BTC" || crypto.Coins[0].Value != 50_000_000 || crypto.Change == nil {
		t.Fatalf("crypto = %+v", crypto)
	}
	if exposure.Change == nil || math.Abs(*exposure.Change-(50_000_000-50_000_000/1.1+200_000-200_000/1.01)) > 1 {
		t.Fatalf("change = %v", exposure.Change)
	}
	if exposure.NetWorth != 100_000_000 || crypto.ShareOfNetWorth == nil || *crypto.ShareOfNetWorth != 50 {
		t.Fatalf("net worth %v share %v", exposure.NetWorth, crypto.ShareOfNetWorth)
	}
	if len(exposure.Issues) != 1 || !strings.Contains(exposure.Issues[0], "ZZZ: no market change (no bars)") {
		t.Fatalf("issues = %q", exposure.Issues)
	}
}

func TestGivenNoMarketWhenComputingExposureThenMovesAreIssuesAndTheAllocationStillCounts(t *testing.T) {
	exposure, err := NewPortfolioService(exposureDB(), nil, nil).Exposure(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if exposure.Change != nil || len(exposure.Issues) != 3 || exposure.Total != 100_000_000 {
		t.Fatalf("exposure = %+v", exposure)
	}
	db := exposureDB()
	db.prefsErr = domain.ErrConflict
	if _, err := NewPortfolioService(db, nil, nil).Exposure(context.Background(), false); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenDailySeriesWhenComputingHedgesThenGoldAndUSDAreReadForAYear(t *testing.T) {
	start := time.Date(2026, 1, 1, 17, 0, 0, 0, time.UTC)
	daily := func(first, last float64) domain.CandleSeries {
		return domain.CandleSeries{Provider: "fmarket", Candles: []domain.Candle{{Time: start, Close: first}, {Time: start.AddDate(0, 6, 0), Close: last}}}
	}
	market := &seriesMarket{candles: map[string]domain.CandleSeries{"SJC": daily(100, 110)}}
	hedges, err := NewPortfolioService(exposureDB(), market, nil).Hedges(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(market.requests)
	if strings.Join(market.requests, ",") != "SJC 1d 380,USDVND 1d 380" || !market.bypassCache {
		t.Fatalf("requests = %v", market.requests)
	}
	if hedges.Gold == nil || hedges.Gold.Change("1Y").ChangePct != 10 || hedges.USD != nil || len(hedges.Issues) != 1 {
		t.Fatalf("hedges = %+v", hedges)
	}
	if hedges.Cash != 100_000_000 {
		t.Fatalf("cash = %v", hedges.Cash)
	}

	offline, err := NewPortfolioService(exposureDB(), nil, nil).Hedges(context.Background(), false)
	if err != nil || offline.Gold != nil || len(offline.Issues) != 2 {
		t.Fatalf("offline = %+v, err = %v", offline, err)
	}
}
