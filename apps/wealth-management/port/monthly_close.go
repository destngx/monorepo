package port

import (
	"apps/wealth-management/domain"
	"context"
)

// MonthlyCloseService is the month-close checklist, the Snapshots tab writes
// and the net worth history built from it (ROADMAP R8, P3).
type MonthlyCloseService interface {
	// MonthlyClose derives the checklist for a "yyyy-mm" month (blank =
	// last month during the first days of a month, else this month).
	MonthlyClose(ctx context.Context, month string, forceFresh bool) (domain.MonthlyClose, error)
	// WriteSnapshot computes this month's or last month's snapshot from a
	// fresh read and writes it, updating the month's row when it exists.
	WriteSnapshot(ctx context.Context, month string) (domain.SnapshotSaveResult, error)
	// NetWorthHistory is every month's net worth, snapshots first and
	// back-calculated estimates elsewhere, with the milestones.
	NetWorthHistory(ctx context.Context, forceFresh bool) (domain.NetWorthHistory, error)
}

// PendingDraftCounter counts the capture drafts waiting for approval.
type PendingDraftCounter interface {
	PendingDraftCount(ctx context.Context) (int, error)
}
