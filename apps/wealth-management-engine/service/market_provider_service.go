package service

import (
	"apps/wealth-management-engine/domain"
	"apps/wealth-management-engine/port"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type marketProviderService struct {
	providers map[string]port.MarketProvider
	routing   domain.MarketRoutingConfig
	cache     port.CacheClient
}

func NewMarketProviderService(providers ...port.MarketProvider) port.MarketProviderService {
	return NewMarketProviderServiceWithRouting(domain.DefaultMarketRoutingConfig(), nil, providers...)
}

func NewMarketProviderServiceWithRouting(
	routing domain.MarketRoutingConfig,
	cache port.CacheClient,
	providers ...port.MarketProvider,
) port.MarketProviderService {
	registry := make(map[string]port.MarketProvider, len(providers))
	for _, provider := range providers {
		registry[strings.ToLower(provider.Name())] = provider
	}

	return &marketProviderService{
		providers: registry,
		routing:   routing,
		cache:     cache,
	}
}

func (s *marketProviderService) Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error) {
	selected, err := s.providerByName(provider)
	if err != nil {
		return domain.MarketProviderHealth{}, err
	}

	return selected.Health(ctx)
}

func (s *marketProviderService) GetTicker(ctx context.Context, symbol string, tickerType domain.TickerType) (domain.Ticker, error) {
	cacheKey := fmt.Sprintf("market:getTicker:%s:%s", tickerType, symbol)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedTicker(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetTicker[tickerType]
	if len(chain) == 0 {
		return domain.Ticker{}, fmt.Errorf("no provider chain configured for ticker type %s", tickerType)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		ticker, err := provider.GetTicker(ctx, symbol, tickerType)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, ticker)
			return ticker, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.Ticker{}, fmt.Errorf("get ticker failed for %s/%s: %s", tickerType, symbol, strings.Join(errors, "; "))
}

func (s *marketProviderService) GetExchangeRate(ctx context.Context, from string, to string) (domain.ExchangeRate, error) {
	pair := strings.ToUpper(from) + "_" + strings.ToUpper(to)
	cacheKey := fmt.Sprintf("market:getExchangeRate:%s", pair)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedExchangeRate(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetExchangeRate[pair]
	if len(chain) == 0 {
		return domain.ExchangeRate{}, fmt.Errorf("no provider chain configured for exchange rate pair %s", pair)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		exchangeRate, err := provider.GetExchangeRate(ctx, from, to)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, exchangeRate)
			return exchangeRate, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.ExchangeRate{}, fmt.Errorf("get exchange rate failed for %s: %s", pair, strings.Join(errors, "; "))
}

func (s *marketProviderService) GetPriceSeries(
	ctx context.Context,
	symbol string,
	seriesType domain.SeriesType,
) (domain.PriceSeries, error) {
	cacheKey := fmt.Sprintf("market:getPriceSeries:%s:%s", seriesType, symbol)
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedPriceSeries(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetPriceSeries[seriesType]
	if len(chain) == 0 {
		return domain.PriceSeries{}, fmt.Errorf("no provider chain configured for series type %s", seriesType)
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		series, err := provider.GetPriceSeries(ctx, symbol, seriesType)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, series)
			return series, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return domain.PriceSeries{}, fmt.Errorf("get price series failed for %s/%s: %s", seriesType, symbol, strings.Join(errors, "; "))
}

func (s *marketProviderService) GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error) {
	cacheKey := "market:getBankInterestRate"
	if !domain.ShouldBypassCache(ctx) {
		if cached, ok, err := s.getCachedBankRates(ctx, cacheKey); err == nil && ok {
			return cached, nil
		}
	}

	chain := s.routing.GetBankInterestRate
	if len(chain) == 0 {
		return nil, fmt.Errorf("no provider chain configured for bank interest rates")
	}

	var errors []string
	for _, name := range chain {
		provider, err := s.providerByName(name)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		rates, err := provider.GetBankInterestRate(ctx)
		if err == nil {
			_ = s.setCache(ctx, cacheKey, rates)
			return rates, nil
		}
		errors = append(errors, fmt.Sprintf("%s: %v", provider.Name(), err))
	}

	return nil, fmt.Errorf("get bank interest rates failed: %s", strings.Join(errors, "; "))
}

func (s *marketProviderService) providerByName(provider string) (port.MarketProvider, error) {
	key := strings.ToLower(strings.TrimSpace(provider))
	selected, ok := s.providers[key]
	if !ok {
		return nil, fmt.Errorf("unknown market provider: %s", provider)
	}

	return selected, nil
}

func (s *marketProviderService) getCachedTicker(ctx context.Context, key string) (domain.Ticker, bool, error) {
	if s.cache == nil {
		return domain.Ticker{}, false, nil
	}

	value, found, err := s.cache.Get(ctx, key)
	if err != nil || !found {
		return domain.Ticker{}, false, err
	}

	var ticker domain.Ticker
	if err := json.Unmarshal([]byte(value), &ticker); err != nil {
		return domain.Ticker{}, false, err
	}

	return ticker, true, nil
}

func (s *marketProviderService) getCachedExchangeRate(ctx context.Context, key string) (domain.ExchangeRate, bool, error) {
	if s.cache == nil {
		return domain.ExchangeRate{}, false, nil
	}

	value, found, err := s.cache.Get(ctx, key)
	if err != nil || !found {
		return domain.ExchangeRate{}, false, err
	}

	var exchangeRate domain.ExchangeRate
	if err := json.Unmarshal([]byte(value), &exchangeRate); err != nil {
		return domain.ExchangeRate{}, false, err
	}

	return exchangeRate, true, nil
}

func (s *marketProviderService) getCachedPriceSeries(ctx context.Context, key string) (domain.PriceSeries, bool, error) {
	if s.cache == nil {
		return domain.PriceSeries{}, false, nil
	}

	value, found, err := s.cache.Get(ctx, key)
	if err != nil || !found {
		return domain.PriceSeries{}, false, err
	}

	var series domain.PriceSeries
	if err := json.Unmarshal([]byte(value), &series); err != nil {
		return domain.PriceSeries{}, false, err
	}

	return series, true, nil
}

func (s *marketProviderService) getCachedBankRates(ctx context.Context, key string) ([]domain.BankRate, bool, error) {
	if s.cache == nil {
		return nil, false, nil
	}

	value, found, err := s.cache.Get(ctx, key)
	if err != nil || !found {
		return nil, false, err
	}

	var rates []domain.BankRate
	if err := json.Unmarshal([]byte(value), &rates); err != nil {
		return nil, false, err
	}

	return rates, true, nil
}

func (s *marketProviderService) setCache(ctx context.Context, key string, payload interface{}) error {
	if s.cache == nil {
		return nil
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return s.cache.Set(ctx, key, string(data), s.routing.CacheTTLSeconds)
}
