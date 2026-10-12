package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
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
			"ETH":   {Symbol: "ETH", Price: 3000, Currency: "USD", Provider: "coingecko"},
			"VESAF": {Symbol: "VESAF", Price: 31000, Currency: "VND", Provider: "fmarket", Timestamp: "2026-10-12T00:00:00Z"},
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
