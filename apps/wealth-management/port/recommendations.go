package port

import (
	"apps/wealth-management/domain"
	"context"
)

// RecommendationService is the recommendations list (ROADMAP R7,
// WM-REC-1..): items from the AI reviews and the engine rules, with the
// status the user sets and the outcome a later review writes.
type RecommendationService interface {
	// List reads the list without writing; ErrValidation for a bad filter.
	List(ctx context.Context, filter domain.RecommendationFilter, forceFresh bool) (domain.RecommendationList, error)
	// Refresh runs the engine rules (adding items whose condition holds and
	// closing those whose condition is gone), then lists.
	Refresh(ctx context.Context, filter domain.RecommendationFilter) (domain.RecommendationList, error)
	// SetStatus marks an item done, dismissed, snoozed or open again when it
	// still matches the ETag (ErrConflict otherwise).
	SetStatus(ctx context.Context, id string, input domain.RecommendationStatusInput) (domain.Recommendation, error)
	// RecentForReview are the items a review of the source checks.
	RecentForReview(ctx context.Context, source string) ([]domain.Recommendation, error)
	// Record adds a review's new recommendations and writes its outcomes.
	Record(ctx context.Context, review domain.ReviewRecommendations) (domain.RecommendationRecord, error)
}
