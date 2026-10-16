package port

import (
	"apps/wealth-management/domain"
	"context"
)

// PortfolioService values the sheet holdings with live prices and explicit
// exchange rates (WM-INV-2/3/4).
type PortfolioService interface {
	Valuation(ctx context.Context, forceFresh bool) (domain.PortfolioValuation, error)
	// Allocation is the one engine allocation (ROADMAP C4): cash, fund
	// certificates and crypto with the Preferences targets and drift, plus
	// money lent as its own line without a target.
	Allocation(ctx context.Context, forceFresh bool) (domain.Allocation, error)
}

// AllocationSource gives the allocation to features that score it, such as
// the health diversification pillar.
type AllocationSource interface {
	Allocation(ctx context.Context, forceFresh bool) (domain.Allocation, error)
}
