package port

import (
	"apps/wealth-management-engine/domain"
	"context"
)

type MarketProvider interface {
	Name() string
	Health(ctx context.Context) (domain.MarketProviderHealth, error)
}

type MarketProviderService interface {
	Health(ctx context.Context, provider string) (domain.MarketProviderHealth, error)
}
