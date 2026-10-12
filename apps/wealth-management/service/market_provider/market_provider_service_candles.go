package market_provider

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// marketLocation is Vietnam time (UTC+7, no DST), used to align 4h bars to
// the HOSE sessions.
var marketLocation = time.FixedZone("ICT", 7*60*60)

const maxCandleLookbackDays = 10 * 365

func (s *marketProviderService) GetCandles(
	ctx context.Context,
	symbol string,
	timeframe domain.Timeframe,
	lookbackDays int,
) (domain.CandleSeries, error) {
	symbol = domain.NormaliseMarketSymbol(symbol)
	if symbol == "" {
		return domain.CandleSeries{}, fmt.Errorf("%w: symbol is required", domain.ErrValidation)
	}
	timeframe, err := domain.ParseTimeframe(string(timeframe))
	if err != nil {
		return domain.CandleSeries{}, err
	}
	if lookbackDays <= 0 {
		lookbackDays = timeframe.AnalysisLookbackDays()
	}
	lookbackDays = min(lookbackDays, maxCandleLookbackDays)

	key := fmt.Sprintf("market:getCandles:%s:%s:%d", timeframe, symbol, lookbackDays)
	if cached, ok := getCached[domain.CandleSeries](ctx, s, key); ok {
		return cached, nil
	}

	now := s.now()
	query := domain.CandleQuery{
		Timeframe: timeframe.SourceTimeframe(),
		From:      now.AddDate(0, 0, -lookbackDays),
		To:        now,
	}
	operation := fmt.Sprintf("get %s candles for %s", timeframe, symbol)
	result, err := route(ctx, s, operation, s.routing.GetCandles, func(provider port.MarketProvider) (domain.CandleSeries, error) {
		capable, ok := provider.(port.CandleProvider)
		if !ok {
			return domain.CandleSeries{}, errCapabilityMissing
		}
		series, err := capable.GetCandles(ctx, symbol, query)
		if err != nil {
			return domain.CandleSeries{}, err
		}
		series.Candles = domain.CleanCandles(series.Candles)
		if len(series.Candles) == 0 {
			return domain.CandleSeries{}, errors.New("returned no candles")
		}
		return series, nil
	})
	if err != nil {
		return serveLastGood[domain.CandleSeries](ctx, s, key, err)
	}

	series := result.value
	if timeframe != query.Timeframe {
		series.Candles = domain.AggregateCandles(series.Candles, timeframe, marketLocation)
	}
	series.Symbol = symbol
	series.Timeframe = timeframe
	series.Provider = result.provider
	series.Fallback = strings.Join(result.failures, "; ")
	s.storeFresh(ctx, key, series, s.routing.CacheTTL.Candles)
	return series, nil
}
