package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// HomeAnalytics are the analytics the inbox reads (port.AnalyticsService has them).
type HomeAnalytics interface {
	Bills(forceFresh bool) (domain.Bills, error)
	BudgetReport(month string, forceFresh bool) (domain.BudgetReport, error)
	GoalPlan(forceFresh bool) (domain.GoalPlan, error)
	MoneyLent(forceFresh bool) (domain.MoneyLent, error)
	IdleCash(ctx context.Context, forceFresh bool) (domain.IdleCash, error)
}

// HomeLedger is the sheet data the inbox reads (port.DatabaseService has it).
type HomeLedger interface {
	ListTransactions(forceFresh bool) ([]domain.Transaction, error)
	ListCategories(forceFresh bool) ([]domain.Category, error)
}

// HomeRecommendations lists the recommendations (port.RecommendationService).
type HomeRecommendations interface {
	List(ctx context.Context, filter domain.RecommendationFilter, forceFresh bool) (domain.RecommendationList, error)
}

// HomeClose derives the month-close checklist (port.MonthlyCloseService).
type HomeClose interface {
	MonthlyClose(ctx context.Context, month string, forceFresh bool) (domain.MonthlyClose, error)
}

// HomeSources feed the inbox. Allocation, Drafts and Close may be nil when
// their feature is not configured (no market data, no quick capture, no
// close): their items are then not built and they are not reported as
// unavailable.
type HomeSources struct {
	Analytics       HomeAnalytics
	Ledger          HomeLedger
	Recommendations HomeRecommendations
	Allocation      port.AllocationSource
	Drafts          port.PendingDraftCounter
	Close           HomeClose
}

type homeService struct {
	sources HomeSources
	log     *logger.Logger
	now     func() time.Time
}

var _ port.HomeService = (*homeService)(nil)

// NewHomeService returns the home service. now may be nil (time.Now).
func NewHomeService(sources HomeSources, log *logger.Logger, now func() time.Time) port.HomeService {
	if now == nil {
		now = time.Now
	}
	return &homeService{sources: sources, log: log, now: now}
}

// homeLoad is one source of the inbox: load fills its input on success.
type homeLoad struct {
	part domain.NeedsYouPart
	load func(ctx context.Context, force bool, in *domain.NeedsYouInputs) error
}

func (s *homeService) loads() []homeLoad {
	src := s.sources
	loads := []homeLoad{
		{domain.NeedsYouPartBills, func(_ context.Context, force bool, in *domain.NeedsYouInputs) error {
			bills, err := src.Analytics.Bills(force)
			in.Bills = &bills
			return err
		}},
		{domain.NeedsYouPartBudget, func(_ context.Context, force bool, in *domain.NeedsYouInputs) error {
			report, err := src.Analytics.BudgetReport("", force)
			if err != nil {
				return err
			}
			categories, err := src.Ledger.ListCategories(force)
			in.Budget, in.Categories = &report, categories
			return err
		}},
		{domain.NeedsYouPartGoals, func(_ context.Context, force bool, in *domain.NeedsYouInputs) error {
			plan, err := src.Analytics.GoalPlan(force)
			in.GoalPlan = &plan
			return err
		}},
		{domain.NeedsYouPartMoneyLent, func(_ context.Context, force bool, in *domain.NeedsYouInputs) error {
			lent, err := src.Analytics.MoneyLent(force)
			in.MoneyLent = &lent
			return err
		}},
		{domain.NeedsYouPartIdleCash, func(ctx context.Context, force bool, in *domain.NeedsYouInputs) error {
			idle, err := src.Analytics.IdleCash(ctx, force)
			in.IdleCash = &idle
			return err
		}},
		{domain.NeedsYouPartTransactions, func(_ context.Context, force bool, in *domain.NeedsYouInputs) error {
			transactions, err := src.Ledger.ListTransactions(force)
			in.Transactions, in.TransactionsLoaded = transactions, true
			return err
		}},
		{domain.NeedsYouPartRecommendations, func(ctx context.Context, force bool, in *domain.NeedsYouInputs) error {
			list, err := src.Recommendations.List(ctx, domain.RecommendationFilter{}, force)
			in.Recommendations = &list
			return err
		}},
	}
	if src.Allocation != nil {
		loads = append(loads, homeLoad{domain.NeedsYouPartAllocation, func(ctx context.Context, force bool, in *domain.NeedsYouInputs) error {
			allocation, err := src.Allocation.Allocation(ctx, force)
			in.Allocation = &allocation
			return err
		}})
	}
	if src.Drafts != nil {
		loads = append(loads, homeLoad{domain.NeedsYouPartDrafts, func(ctx context.Context, _ bool, in *domain.NeedsYouInputs) error {
			pending, err := src.Drafts.PendingDraftCount(ctx)
			in.PendingDrafts = &pending
			return err
		}})
	}
	if src.Close != nil {
		loads = append(loads, homeLoad{domain.NeedsYouPartClose, func(ctx context.Context, force bool, in *domain.NeedsYouInputs) error {
			checklist, err := src.Close.MonthlyClose(ctx, "", force)
			in.Close = &checklist
			return err
		}})
	}
	return loads
}

// NeedsYou loads every source at once, each into its own inputs, then
// keeps the ones that loaded. A failed source is logged, left out and
// listed as unavailable.
func (s *homeService) NeedsYou(ctx context.Context, forceFresh bool) (domain.NeedsYou, error) {
	loads := s.loads()
	parts := make([]domain.NeedsYouInputs, len(loads))
	errs := make([]error, len(loads))
	var wg sync.WaitGroup
	for i, load := range loads {
		wg.Go(func() {
			errs[i] = load.load(ctx, forceFresh, &parts[i])
		})
	}
	wg.Wait()

	inputs := domain.NeedsYouInputs{Today: s.now()}
	var failures []error
	for i, load := range loads {
		if errs[i] != nil {
			s.log.LogError(ctx, fmt.Sprintf("needs you: %s unavailable, its items are left out", load.part), errs[i])
			inputs.Unavailable = append(inputs.Unavailable, domain.NeedsYouMissing(load.part))
			failures = append(failures, fmt.Errorf("%s: %w", load.part, errs[i]))
			continue
		}
		mergeHomeInputs(&inputs, parts[i])
	}
	if len(failures) == len(loads) {
		return domain.NeedsYou{}, fmt.Errorf("needs you: no source could be loaded: %w", errors.Join(failures...))
	}
	return domain.BuildNeedsYou(inputs), nil
}

// mergeHomeInputs copies the fields one source filled.
func mergeHomeInputs(into *domain.NeedsYouInputs, part domain.NeedsYouInputs) {
	if part.PendingDrafts != nil {
		into.PendingDrafts = part.PendingDrafts
	}
	if part.Bills != nil {
		into.Bills = part.Bills
	}
	if part.Budget != nil {
		into.Budget, into.Categories = part.Budget, part.Categories
	}
	if part.MoneyLent != nil {
		into.MoneyLent = part.MoneyLent
	}
	if part.GoalPlan != nil {
		into.GoalPlan = part.GoalPlan
	}
	if part.Allocation != nil {
		into.Allocation = part.Allocation
	}
	if part.IdleCash != nil {
		into.IdleCash = part.IdleCash
	}
	if part.TransactionsLoaded {
		into.Transactions, into.TransactionsLoaded = part.Transactions, true
	}
	if part.Recommendations != nil {
		into.Recommendations = part.Recommendations
	}
	if part.Close != nil {
		into.Close = part.Close
	}
}
