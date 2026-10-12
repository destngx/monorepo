package domain

import (
	"errors"
	"testing"
	"time"
)

func TestGivenTimeframeTextWhenParseTimeframeThenAcceptsKnownValuesOnly(t *testing.T) {
	for _, raw := range []string{"1h", "4H", " 1d ", "1w"} {
		if _, err := ParseTimeframe(raw); err != nil {
			t.Fatalf("ParseTimeframe(%q) error: %v", raw, err)
		}
	}
	if _, err := ParseTimeframe("15m"); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected a validation error, got %v", err)
	}
	if Timeframe4H.SourceTimeframe() != Timeframe1H || Timeframe1D.SourceTimeframe() != Timeframe1D {
		t.Fatal("4h must be built from 1h and others fetched directly")
	}
}

func TestGivenRawCandlesWhenCleanCandlesThenSortsDropsEmptyAndDeduplicates(t *testing.T) {
	base := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	cleaned := CleanCandles([]Candle{
		{Time: base.Add(2 * time.Hour), Close: 3},
		{Time: base, Close: 1},
		{Time: base.Add(time.Hour), Close: 0},
		{Time: base.Add(2 * time.Hour), Open: 2, High: 4, Low: 2, Close: 3.5},
		{Close: 9},
	})
	if len(cleaned) != 2 {
		t.Fatalf("expected two candles, got %#v", cleaned)
	}
	if cleaned[0].Close != 1 || cleaned[0].Open != 1 || cleaned[0].High != 1 || cleaned[0].Low != 1 {
		t.Fatalf("expected missing OHL filled from close, got %#v", cleaned[0])
	}
	if cleaned[1].Close != 3.5 {
		t.Fatalf("expected the later duplicate to win, got %#v", cleaned[1])
	}
}

func TestGivenHourlyCandlesWhenAggregateCandlesToFourHourThenBucketsByLocalTime(t *testing.T) {
	location := time.FixedZone("ICT", 7*60*60)
	at := func(hour int) time.Time { return time.Date(2026, 10, 9, hour, 0, 0, 0, location) }
	bars := AggregateCandles([]Candle{
		{Time: at(9), Open: 10, High: 11, Low: 9, Close: 10.5, Volume: 1},
		{Time: at(10), Open: 10.5, High: 12, Low: 10, Close: 11, Volume: 2},
		{Time: at(11), Open: 11, High: 11.5, Low: 8, Close: 9, Volume: 3},
		{Time: at(13), Open: 9, High: 9.5, Low: 8.5, Close: 9.2, Volume: 4},
	}, Timeframe4H, location)
	if len(bars) != 2 {
		t.Fatalf("expected two bars, got %#v", bars)
	}
	if !bars[0].Time.Equal(at(8)) || bars[0].Open != 10 || bars[0].High != 12 || bars[0].Low != 8 || bars[0].Close != 9 || bars[0].Volume != 6 {
		t.Fatalf("unexpected first bar: %#v", bars[0])
	}
	if !bars[1].Time.Equal(at(12)) || bars[1].Close != 9.2 {
		t.Fatalf("unexpected second bar: %#v", bars[1])
	}
}

func TestGivenSymbolsWhenNormaliseMarketSymbolThenUppercasesAndDropsCaret(t *testing.T) {
	if got := NormaliseMarketSymbol(" ^vnIndex "); got != "VNINDEX" {
		t.Fatalf("got %q", got)
	}
}

func TestGivenHourlyCandlesWhenAggregateCandlesToDayAndWeekThenBucketsByLocalDateAndMonday(t *testing.T) {
	location := time.FixedZone("ICT", 7*60*60)
	// Thursday 9 Oct, Friday 10 Oct and Monday 13 Oct 2026.
	hours := []Candle{
		{Time: time.Date(2026, 10, 9, 9, 0, 0, 0, location), Open: 1, High: 2, Low: 1, Close: 2},
		{Time: time.Date(2026, 10, 9, 14, 0, 0, 0, location), Open: 2, High: 3, Low: 2, Close: 3},
		{Time: time.Date(2026, 10, 10, 9, 0, 0, 0, location), Open: 3, High: 4, Low: 3, Close: 4},
		{Time: time.Date(2026, 10, 13, 9, 0, 0, 0, location), Open: 4, High: 5, Low: 4, Close: 5},
	}
	days := AggregateCandles(hours, Timeframe1D, location)
	if len(days) != 3 || days[0].Close != 3 || days[0].High != 3 || !days[0].Time.Equal(time.Date(2026, 10, 9, 0, 0, 0, 0, location)) {
		t.Fatalf("unexpected daily bars: %#v", days)
	}
	weeks := AggregateCandles(hours, Timeframe1W, location)
	if len(weeks) != 2 || !weeks[0].Time.Equal(time.Date(2026, 10, 5, 0, 0, 0, 0, location)) || weeks[0].Close != 4 || weeks[1].Open != 4 {
		t.Fatalf("unexpected weekly bars: %#v", weeks)
	}
}
