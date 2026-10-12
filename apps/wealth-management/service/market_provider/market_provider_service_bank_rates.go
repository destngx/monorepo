package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
)

func (s *marketProviderService) GetBankInterestRate(ctx context.Context) ([]domain.BankRate, error) {
	return cachedRoute(ctx, s, "market:getBankInterestRate", s.routing.CacheTTL.BankRates, "get bank interest rates", s.routing.GetBankInterestRate, func(provider port.MarketProvider) ([]domain.BankRate, error) {
		capable, ok := provider.(port.BankRateProvider)
		if !ok {
			return nil, errCapabilityMissing
		}
		return capable.GetBankInterestRate(ctx)
	})
}
