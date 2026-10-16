package service

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"slices"
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
	navs   port.FundNAVSource
	now    func() time.Time
}

// PortfolioOption configures the portfolio service.
type PortfolioOption func(*portfolioService)

// WithFundNAVs gives fund performance the NAV histories (WM-INV-21).
func WithFundNAVs(navs port.FundNAVSource) PortfolioOption {
	return func(s *portfolioService) { s.navs = navs }
}

// NewPortfolioService values holdings with live prices from market; with a
// nil market only sheet values count and the rest are reported as issues.
func NewPortfolioService(db port.DatabaseService, market port.MarketProviderService, now func() time.Time, options ...PortfolioOption) port.PortfolioService {
	if now == nil {
		now = time.Now
	}
	service := &portfolioService{db: db, market: market, now: now}
	for _, option := range options {
		option(service)
	}
	return service
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

	if s.market == nil {
		for _, lookup := range priceLookups(assets) {
			in.PriceIssues[lookup.key] = errMarketNotConfigured
		}
		for _, currency := range []domain.Currency{domain.CurrencyUSD, domain.CurrencyUSDT} {
			in.RateIssues[currency] = errMarketNotConfigured
		}
		return domain.ValuePortfolio(in), nil
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
				Source:    ticker.Source,
			}
		})
	}
	group.Wait()

	return domain.ValuePortfolio(in), nil
}

// errMarketNotConfigured explains missing live prices without a market server.
const errMarketNotConfigured = "market data is not configured"

// Allocation combines the valuation's holdings, the account balances, money
// lent from the transactions and the targets from the Preferences tab.
func (s *portfolioService) Allocation(ctx context.Context, forceFresh bool) (domain.Allocation, error) {
	valuation, err := s.Valuation(ctx, forceFresh)
	if err != nil {
		return domain.Allocation{}, err
	}
	ledger, err := s.db.ReadLedger(forceFresh)
	if err != nil {
		return domain.Allocation{}, fmt.Errorf("read ledger: %w", err)
	}
	prefs, err := s.db.ListPreferences(forceFresh)
	if err != nil {
		return domain.Allocation{}, fmt.Errorf("read preferences: %w", err)
	}
	in := domain.AllocationInputsFromValuation(valuation)
	in.Accounts, in.Transactions = ledger.Accounts, ledger.Transactions
	in.Targets = prefs.AllocationTargets
	in.Issues = append(in.Issues, prefs.Issues...)
	return domain.ComputeAllocation(in), nil
}

// errNAVNotConfigured explains missing NAV histories.
const errNAVNotConfigured = "fund NAV history is not configured (market data server)"

// FundPerformance reads the fund holdings with their History, the NAV
// history of each fund code and the bank rates, then computes gains,
// returns and the deposit comparison. Market failures become issues on
// the answer; only sheet failures fail the call.
func (s *portfolioService) FundPerformance(ctx context.Context, forceFresh bool) (domain.FundPerformance, error) {
	assets, err := s.db.GetInvestmentAssets(forceFresh)
	if err != nil {
		return domain.FundPerformance{}, fmt.Errorf("read holdings: %w", err)
	}
	ctx = domain.WithBypassCache(ctx, forceFresh)
	in := domain.FundPerformanceInputs{
		Holdings:  assets.Funds.Items,
		NAVs:      map[string][]domain.NavPoint{},
		NAVIssues: map[string]string{},
		Now:       s.now(),
	}
	codes := []string{}
	for _, holding := range assets.Funds.Items {
		if code := domain.FundKey(holding.Symbol); code != "" && !slices.Contains(codes, code) {
			codes = append(codes, code)
		}
	}

	var mu sync.Mutex
	group := newBoundedGroup(maxPriceLookups)
	for _, code := range codes {
		if s.navs == nil {
			in.NAVIssues[code] = errNAVNotConfigured
			continue
		}
		group.Go(func() {
			history, err := s.navs.GetFundNAVHistory(ctx, code)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				in.NAVIssues[code] = err.Error()
				return
			}
			in.NAVs[code] = history.Points
		})
	}
	if s.market == nil {
		in.DepositIssue = "bank rates unavailable: " + errMarketNotConfigured
	} else {
		group.Go(func() {
			rates, err := s.market.GetBankInterestRate(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				in.DepositIssue = "bank rates unavailable: " + err.Error()
				return
			}
			if rate, ok := domain.DepositRateFromBankRates(rates.Rates); ok {
				in.Deposit = &rate
				return
			}
			in.DepositIssue = "no bank published a savings rate"
		})
	}
	group.Wait()
	return domain.ComputeFundPerformance(in), nil
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
