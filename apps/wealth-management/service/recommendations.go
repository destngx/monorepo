package service

import (
	"apps/wealth-management/adapter/logger"
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"fmt"
	"log/slog"
	"time"
)

// recommendationService is the recommendations list (ROADMAP R7): reads and
// status changes go to the Recommendations tab, the engine rules read the
// analytics, and every write is planned on a fresh read under the store's
// write lock.
type recommendationService struct {
	db         port.DatabaseService
	analytics  port.AnalyticsService
	allocation port.AllocationSource
	log        *logger.Logger
	now        func() time.Time
}

var _ port.RecommendationService = (*recommendationService)(nil)

// NewRecommendationService returns the list service. allocation may be nil:
// the rebalance rule is then not evaluated. now may be nil (time.Now).
func NewRecommendationService(db port.DatabaseService, analytics port.AnalyticsService, allocation port.AllocationSource, log *logger.Logger, now func() time.Time) port.RecommendationService {
	if now == nil {
		now = time.Now
	}
	return &recommendationService{db: db, analytics: analytics, allocation: allocation, log: log, now: now}
}

func (s *recommendationService) today() time.Time {
	return domain.Today(s.now())
}

func (s *recommendationService) List(_ context.Context, filter domain.RecommendationFilter, forceFresh bool) (domain.RecommendationList, error) {
	if err := filter.Validate(); err != nil {
		return domain.RecommendationList{}, err
	}
	sheet, err := s.db.ReadRecommendations(forceFresh)
	if err != nil {
		return domain.RecommendationList{}, err
	}
	return domain.BuildRecommendationList(sheet, filter, s.today()), nil
}

func (s *recommendationService) Refresh(ctx context.Context, filter domain.RecommendationFilter) (domain.RecommendationList, error) {
	if err := filter.Validate(); err != nil {
		return domain.RecommendationList{}, err
	}
	evaluated, findings := domain.EvaluateRecommendationRules(s.ruleInputs(ctx))
	today := s.today()
	write, err := s.db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.PlanRuleRecommendations(sheet, evaluated, findings, today), nil
	})
	if err != nil {
		return domain.RecommendationList{}, err
	}
	return domain.BuildRecommendationList(write.Sheet, filter, today), nil
}

// ruleInputs loads what the rules read. An input that fails is logged and
// left nil, so its rule is skipped this time and its items stay as they are.
func (s *recommendationService) ruleInputs(ctx context.Context) domain.RuleInputs {
	inputs := domain.RuleInputs{}
	if s.allocation != nil {
		if allocation, err := s.allocation.Allocation(ctx, false); err != nil {
			s.log.LogError(ctx, "recommendations: allocation unavailable, rebalance rule skipped", err)
		} else {
			inputs.Allocation = &allocation
		}
	}
	if plan, err := s.analytics.GoalPlan(false); err != nil {
		s.log.LogError(ctx, "recommendations: goal plan unavailable, goal rule skipped", err)
	} else {
		inputs.GoalPlan = &plan
	}
	if lent, err := s.analytics.MoneyLent(false); err != nil {
		s.log.LogError(ctx, "recommendations: money lent unavailable, lending rule skipped", err)
	} else {
		inputs.MoneyLent = &lent
	}
	if idle, err := s.analytics.IdleCash(ctx, false); err != nil {
		s.log.LogError(ctx, "recommendations: idle cash unavailable, idle cash rule skipped", err)
	} else {
		inputs.IdleCash = &idle
	}
	return inputs
}

func (s *recommendationService) SetStatus(_ context.Context, id string, input domain.RecommendationStatusInput) (domain.Recommendation, error) {
	if err := input.Validate(); err != nil {
		return domain.Recommendation{}, err
	}
	today := s.today()
	write, err := s.db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		return domain.PlanStatusChange(sheet, id, input, today)
	})
	if err != nil {
		return domain.Recommendation{}, err
	}
	items := write.Items(write.Updated)
	if len(items) != 1 {
		return domain.Recommendation{}, errNotReadBack(id)
	}
	return items[0], nil
}

func (s *recommendationService) RecentForReview(_ context.Context, source string) ([]domain.Recommendation, error) {
	sheet, err := s.db.ReadRecommendations(false)
	if err != nil {
		return nil, err
	}
	return domain.RecentForReview(sheet, source, s.today()), nil
}

func (s *recommendationService) Record(ctx context.Context, review domain.ReviewRecommendations) (domain.RecommendationRecord, error) {
	record := domain.RecommendationRecord{Created: []domain.Recommendation{}, Checked: []domain.Recommendation{}}
	if len(review.Proposals) == 0 && len(review.Outcomes) == 0 {
		return record, nil
	}
	today := s.today()
	write, err := s.db.UpdateRecommendations(func(sheet domain.RecommendationSheet) (domain.RecommendationChanges, error) {
		changes, duplicates, ignored := domain.PlanReviewRecord(sheet, review, today)
		record.Duplicates, record.IgnoredOutcomes = duplicates, ignored
		return changes, nil
	})
	if err != nil {
		return domain.RecommendationRecord{}, err
	}
	record.Created = write.Items(write.Added)
	record.Checked = write.Items(write.Updated)
	if record.IgnoredOutcomes > 0 {
		s.log.LogApplicationEvent(ctx, "recommendations: outcomes for ids the review was not given were ignored",
			slog.String("source", review.Source), slog.Int("ignored", record.IgnoredOutcomes))
	}
	return record, nil
}

func errNotReadBack(id string) error {
	return fmt.Errorf("recommendation %s was written but could not be read back; reload the list", id)
}
