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
	// FundPerformance is each fund holding's gain and returns from its
	// History priced at the NAV history, compared with a bank deposit
	// (WM-INV-21..23).
	FundPerformance(ctx context.Context, forceFresh bool) (domain.FundPerformance, error)
	// Exposure is the Investments headline (ROADMAP M1, M2): the volatile
	// share of the allocation, today's change from coin and fund NAV moves,
	// the held coins and the crypto drawdown scenarios against net worth.
	Exposure(ctx context.Context, forceFresh bool) (domain.Exposure, error)
	// Hedges is the gold and USD/VND trend against the VND cash share
	// (ROADMAP M4).
	Hedges(ctx context.Context, forceFresh bool) (domain.Hedges, error)
}

// AllocationSource gives the allocation to features that score it, such as
// the health diversification pillar.
type AllocationSource interface {
	Allocation(ctx context.Context, forceFresh bool) (domain.Allocation, error)
}
