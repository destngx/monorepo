package port

import (
	"apps/wealth-management/domain"
	"context"
)

type FmarketService interface {
	RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error)
}
