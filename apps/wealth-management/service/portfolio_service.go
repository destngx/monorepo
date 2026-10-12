package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// maxPriceLookups bounds concurrent quote calls so a large portfolio does
// not trip the public CoinGecko rate limit.
const maxPriceLookups = 4

type portfolioService struct {
	db     port.DatabaseService
	market port.MarketProviderService
	now    func() time.Time
}

func NewPortfolioService(db port.DatabaseService, market port.MarketProviderService, now func() time.Time) port.PortfolioService {
	if now == nil {
		now = time.Now
	}
	return &portfolioService{db: db, market: market, now: now}
}

// Valuation reads holdings and accounts from the sheet, prices every holding
// with a symbol (crypto in USD via CoinGecko, funds by fmarket NAV) and
// converts with live USD and USDT rates. Market failures become issues on
// the answer; only sheet failures fail the call.
func (s *portfolioService) Valuation(ctx context.Context, forceFresh bool) (domain.PortfolioValuation, error) {
	assets, err := s.db.GetInvestmentAssets(forceFresh)
	if err != nil {
		return domain.PortfolioValuation{}, fmt.Errorf("read holdings: %w", err)
	}
	accounts, err := s.db.ListAccounts(forceFresh)
	if err != nil {
		return domain.PortfolioValuation{}, fmt.Errorf("read accounts: %w", err)
	}

	ctx = domain.WithBypassCache(ctx, forceFresh)
	in := domain.PortfolioInputs{
		Assets:      assets,
		Accounts:    accounts,
		Prices:      map[string]domain.LivePrice{},
		PriceIssues: map[string]string{},
		Rates:       map[domain.Currency]domain.ExchangeRate{},
		RateIssues:  map[domain.Currency]string{},
		Now:         s.now(),
	}

	var mu sync.Mutex
	group := newBoundedGroup(maxPriceLookups)

	for _, currency := range []domain.Currency{domain.CurrencyUSD, domain.CurrencyUSDT} {
		group.Go(func() {
			rate, err := s.market.GetExchangeRate(ctx, string(currency), string(domain.CurrencyVND))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				in.RateIssues[currency] = err.Error()
				return
			}
			in.Rates[currency] = rate
		})
	}
	for _, lookup := range priceLookups(assets) {
		group.Go(func() {
			ticker, err := s.market.GetTicker(ctx, lookup.symbol, lookup.tickerType)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				in.PriceIssues[lookup.key] = err.Error()
				return
			}
			in.Prices[lookup.key] = domain.LivePrice{
				Price:     ticker.Price,
				Currency:  domain.Currency(strings.ToUpper(ticker.Currency)),
				Provider:  ticker.Provider,
				Timestamp: ticker.Timestamp,
			}
		})
	}
	group.Wait()

	return domain.ValuePortfolio(in), nil
}

type priceLookup struct {
	key        string
	symbol     string
	tickerType domain.TickerType
}

// priceLookups lists each distinct holding symbol once.
func priceLookups(assets domain.InvestmentAssets) []priceLookup {
	seen := map[string]bool{}
	lookups := []priceLookup{}
	add := func(section string, tickerType domain.TickerType, holdings []domain.Holding) {
		for _, holding := range holdings {
			symbol := strings.ToUpper(strings.TrimSpace(holding.Symbol))
			key := domain.PriceKey(section, symbol)
			if symbol == "" || seen[key] {
				continue
			}
			seen[key] = true
			lookups = append(lookups, priceLookup{key: key, symbol: symbol, tickerType: tickerType})
		}
	}
	add(domain.PortfolioSectionCrypto, domain.TickerTypeCrypto, assets.Crypto.Items)
	add(domain.PortfolioSectionFunds, domain.TickerTypeIFC, assets.Funds.Items)
	return lookups
}
