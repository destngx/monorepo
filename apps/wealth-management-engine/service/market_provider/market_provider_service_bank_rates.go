package market_provider

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"strings"
)

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
