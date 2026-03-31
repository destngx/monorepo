package vnstock

import (
	"apps/wealth-management-engine/domain"
	"context"
	"fmt"
	"strings"
)

func (p *Provider) GetExchangeRate(_ context.Context, from string, to string) (domain.ExchangeRate, error) {
	upperFrom := strings.ToUpper(from)
	upperTo := strings.ToUpper(to)
	if upperFrom != "USD" || upperTo != "VND" {
		return domain.ExchangeRate{}, fmt.Errorf("vnstock supports fallback exchange rate only for USD_VND")
	}

	return domain.ExchangeRate{}, fmt.Errorf("vnstock exchange-rate endpoint is unavailable in current API surface")
}
