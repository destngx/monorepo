package port

import (
	"apps/wealth-management/domain"
	"context"
)

// HomeService is the home page (ROADMAP R5).
type HomeService interface {
	// NeedsYou ranks what the user should act on (WM-HOME-1). A source that
	// fails is logged and listed as unavailable; only when every source
	// fails is the error returned.
	NeedsYou(ctx context.Context, forceFresh bool) (domain.NeedsYou, error)
}
