package port

import (
	"apps/wealth-management-engine/domain"
	"context"
)

type ToolIntegrationService interface {
	RunConversation(ctx context.Context, prompt string) (domain.ToolConversation, error)
}

type BalanceTool interface {
	GetBalance(ctx context.Context, account string) (float64, error)
}

type MarketAnalysisSkill interface {
	Analyze(ctx context.Context, ticker string) (string, error)
}
