package port

import (
	"apps/wealth-management/domain"
	"context"
)

// MarketAnalysisService computes the deterministic market views from
// candles: the VN market pulse (WM-INV-11) and the ticker deep-dive
// technicals, confluence, seasonality and risk (WM-INV-12/13/14/16).
type MarketAnalysisService interface {
	Pulse(ctx context.Context, timeframe domain.Timeframe) (domain.MarketPulse, error)
	Analysis(ctx context.Context, symbol string, timeframe domain.Timeframe) (domain.MarketAnalysis, error)
}
