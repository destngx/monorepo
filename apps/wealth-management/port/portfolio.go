package port

import (
	"apps/wealth-management/domain"
	"context"
)

// PortfolioService values the sheet holdings with live prices and explicit
// exchange rates (WM-INV-2/3/4).
type PortfolioService interface {
	Valuation(ctx context.Context, forceFresh bool) (domain.PortfolioValuation, error)
}
