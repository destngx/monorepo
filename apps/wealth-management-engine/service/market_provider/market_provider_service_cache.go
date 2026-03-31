package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"encoding/json"
)

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
