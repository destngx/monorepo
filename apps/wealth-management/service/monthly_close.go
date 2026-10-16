package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// MonthlyCloseService backs the monthly close page (ROADMAP R8), the
// Snapshots tab writes and the net worth history (P3). The checklist reads
// the drafts, budget, goal plan, money lent and recommendations as optional
// context: one that
// cannot be read is logged and its item shows as unavailable, so the rest of
// the checklist still works.
type MonthlyCloseService struct {
	db         port.DatabaseService
	analytics  port.AnalyticsService
	allocation port.AllocationSource
	// recommendations is the recommendations list (ROADMAP R7); optional.
	recommendations port.RecommendationService
	drafts          atomic.Pointer[draftCounterRef]
	log             *logger.Logger
	now             func() time.Time
}

type draftCounterRef struct{ counter port.PendingDraftCounter }

var _ port.MonthlyCloseService = (*MonthlyCloseService)(nil)

// NewMonthlyCloseService builds the service. allocation values the fund and
// crypto holdings for snapshots; nil leaves them at the account balances.
// recommendations counts the open recommendations; nil makes that item
// unavailable.
func NewMonthlyCloseService(db port.DatabaseService, analytics port.AnalyticsService, allocation port.AllocationSource, recommendations port.RecommendationService, log *logger.Logger, now func() time.Time) *MonthlyCloseService {
	if now == nil {
		now = time.Now
	}
	return &MonthlyCloseService{db: db, analytics: analytics, allocation: allocation, recommendations: recommendations, log: log, now: now}
}

// SetDraftCounter wires quick capture, which starts after the sheet
// services; until then the drafts item is unavailable.
func (s *MonthlyCloseService) SetDraftCounter(counter port.PendingDraftCounter) {
	if counter == nil {
		s.drafts.Store(nil)
		return
	}
	s.drafts.Store(&draftCounterRef{counter: counter})
}

func (s *MonthlyCloseService) today() time.Time {
	return domain.Today(s.now())
}

func (s *MonthlyCloseService) MonthlyClose(ctx context.Context, month string, forceFresh bool) (domain.MonthlyClose, error) {
	today := s.today()
	target, err := domain.ParseCloseMonth(month, today)
	if err != nil {
		return domain.MonthlyClose{}, err
	}
	ledger, err := s.db.ReadLedger(forceFresh)
	if err != nil {
		return domain.MonthlyClose{}, err
	}
	snapshots, err := s.db.ReadSnapshots(forceFresh)
	if err != nil {
		return domain.MonthlyClose{}, err
	}
	in := domain.CloseInputs{Month: target, Today: today, Transactions: ledger.Transactions, Snapshots: snapshots}

	if ref := s.drafts.Load(); ref == nil {
		in.DraftsIssue = "quick capture is not available (it needs the Google Sheet and the AI settings)"
	} else if count, err := ref.counter.PendingDraftCount(ctx); err != nil {
		s.log.LogError(ctx, "monthly close: pending drafts unavailable", err)
		in.DraftsIssue = err.Error()
	} else {
		in.PendingDrafts = &count
	}
	if report, err := s.analytics.BudgetReport(domain.MonthKey(target), forceFresh); err != nil {
		s.log.LogError(ctx, "monthly close: budget unavailable", err)
		in.BudgetIssue = err.Error()
	} else {
		in.Budget = &report
	}
	if plan, err := s.analytics.GoalPlan(forceFresh); err != nil {
		s.log.LogError(ctx, "monthly close: goal plan unavailable", err)
		in.GoalsIssue = err.Error()
	} else {
		in.Goals = &plan
	}
	if lent, err := s.analytics.MoneyLent(forceFresh); err != nil {
		s.log.LogError(ctx, "monthly close: money lent unavailable", err)
		in.MoneyLentIssue = err.Error()
	} else {
		in.MoneyLent = &lent
	}
	if s.recommendations == nil {
		in.RecommendationsIssue = "the recommendations list is not available"
	} else if list, err := s.recommendations.List(ctx, domain.RecommendationFilter{}, forceFresh); err != nil {
		s.log.LogError(ctx, "monthly close: recommendations unavailable", err)
		in.RecommendationsIssue = err.Error()
	} else {
		in.Recommendations = &list.Summary
	}
	return domain.ComputeMonthlyClose(in), nil
}

// WriteSnapshot reads the ledger and the holdings afresh, so the row holds
// today's balances and prices, and writes the month's row.
func (s *MonthlyCloseService) WriteSnapshot(ctx context.Context, month string) (domain.SnapshotSaveResult, error) {
	today := s.today()
	target, err := domain.SnapshotMonth(month, today)
	if err != nil {
		return domain.SnapshotSaveResult{}, err
	}
	ledger, err := s.db.ReadLedger(true)
	if err != nil {
		return domain.SnapshotSaveResult{}, err
	}
	in := domain.SnapshotInputs{Accounts: ledger.Accounts, Transactions: ledger.Transactions, Month: target, Today: today}
	if s.allocation != nil {
		allocation, err := s.allocation.Allocation(ctx, true)
		if err != nil {
			return domain.SnapshotSaveResult{}, fmt.Errorf("value holdings for the snapshot: %w", err)
		}
		in.Funds = allocation.Line(domain.AllocationFunds).Value
		in.CryptoHoldings = allocation.Line(domain.AllocationCrypto).Value
	}
	return s.db.SaveSnapshot(domain.ComputeMonthSnapshot(in))
}

func (s *MonthlyCloseService) NetWorthHistory(_ context.Context, forceFresh bool) (domain.NetWorthHistory, error) {
	ledger, err := s.db.ReadLedger(forceFresh)
	if err != nil {
		return domain.NetWorthHistory{}, err
	}
	snapshots, err := s.db.ReadSnapshots(forceFresh)
	if err != nil {
		return domain.NetWorthHistory{}, err
	}
	return domain.ComputeNetWorthHistory(domain.NetWorthHistoryInputs{
		Accounts: ledger.Accounts, Transactions: ledger.Transactions, Snapshots: snapshots, Today: s.today(),
	}), nil
}
