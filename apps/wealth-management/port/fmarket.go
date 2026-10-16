package port

import (
	"apps/wealth-management/domain"
	"context"
)

type FmarketService interface {
	RunAction(ctx context.Context, request domain.FmarketRequest, forceFresh bool) (any, error)
}

// FundNAVSource gives a fund's full published NAV history by its code
// (the market data server's fmarket NAV history).
type FundNAVSource interface {
	GetFundNAVHistory(ctx context.Context, code string) (domain.FundNAVHistory, error)
}
