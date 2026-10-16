package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func hourlyBars(start time.Time, closes ...float64) []Candle {
	candles := make([]Candle, len(closes))
	for i, close := range closes {
		candles[i] = Candle{Time: start.Add(time.Duration(i) * time.Hour), Close: close}
	}
	return candles
}

func TestGivenHourlyBarsWhenComputingTheCryptoDayMoveThenTheLatestCloseIsComparedWithTheCloseADayBefore(t *testing.T) {
	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	closes := make([]float64, 26)
	for i := range closes {
		closes[i] = 100
	}
	closes[1] = 200 // 24 hours before the last bar
	closes[25] = 220
	move, err := CryptoDayMove(CandleSeries{Provider: "yahoo", Candles: hourlyBars(start, closes...)})
	if err != nil {
		t.Fatal(err)
	}
	if move.Basis != MoveBasisDay || move.Previous != 200 || move.Price != 220 || move.ChangePct != 10 {
		t.Fatalf("move = %+v", move)
	}
	if move.Since != "2026-10-15T01:00:00Z" || move.AsOf != "2026-10-16T01:00:00Z" || move.Source == nil || move.Source.Provider != "yahoo" {
		t.Fatalf("times or source = %+v", move)
	}
}

func TestGivenLessThanADayOfBarsWhenComputingTheCryptoDayMoveThenThereIsNoMove(t *testing.T) {
	start := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	if _, err := CryptoDayMove(CandleSeries{Candles: hourlyBars(start, 1, 2, 3)}); !errors.Is(err, errNoMove) {
		t.Fatalf("err = %v", err)
	}
	// a gap: the only earlier bar is two days back
	gap := []Candle{{Time: start, Close: 1}, {Time: start.Add(48 * time.Hour), Close: 2}}
	if _, err := CryptoDayMove(CandleSeries{Candles: gap}); !errors.Is(err, errNoMove) {
		t.Fatalf("err = %v", err)
	}
}

func TestGivenANAVHistoryWhenComputingTheFundMoveThenTheLatestNAVIsComparedWithThePreviousOne(t *testing.T) {
	move, err := FundNAVMove([]NavPoint{{Date: "2026-10-14", NAV: 10_000}, {Date: "2026-10-16", NAV: 10_100}, {Date: "2026-10-15", NAV: 10_000}, {Date: "bad"}})
	if err != nil {
		t.Fatal(err)
	}
	if move.Basis != MoveBasisLastNAV || move.Since != "2026-10-15" || move.AsOf != "2026-10-16" || move.ChangePct != 1 {
		t.Fatalf("move = %+v", move)
	}
	if _, err := FundNAVMove([]NavPoint{{Date: "2026-10-16", NAV: 1}}); !errors.Is(err, errNoMove) {
		t.Fatalf("err = %v", err)
	}
}

// exposureInputs: cash 60M, funds 30M, crypto 100M (holdings), money lent
// 10M; net worth 150M. Invented figures.
func exposureInputs() ExposureInputs {
	btc, eth, fund := 70_000_000.0, 30_000_000.0, 30_000_000.0
	one, two, ten := 1.0, 2.0, 10.0
	holdings := []ValuedHolding{
		{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Exchange A", Symbol: "btc", Quantity: &one}, ValueVND: &btc, LivePrice: &LivePrice{Price: 2660, Currency: CurrencyUSD}},
		{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Wallet B", Symbol: "ETH", Quantity: &two}, ValueVND: &eth},
		{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Wallet B", Symbol: "ZZZ", Quantity: &one}, Issue: "no live price: unknown"},
		{Section: PortfolioSectionFunds, Holding: Holding{Platform: "Fund platform", Symbol: "FUND-A", Quantity: &ten}, ValueVND: &fund},
	}
	allocation := ComputeAllocation(AllocationInputs{
		Accounts:       []Account{{Name: "Bank", Type: AccountTypeBank, Balance: 60_000_000}},
		Transactions:   []Transaction{lendingTxn("row-2", "2026-01-02", "Person A", 10_000_000, 0)},
		Funds:          30_000_000,
		CryptoHoldings: 100_000_000,
	})
	return ExposureInputs{
		Allocation:   allocation,
		Valuation:    PortfolioValuation{Holdings: holdings, Totals: PortfolioTotals{CryptoHoldings: 100_000_000}},
		BalanceSheet: BalanceSheet{NetWorth: 150_000_000},
		CryptoMoves:  map[string]PriceMove{"BTC": {Basis: MoveBasisDay, ChangePct: 40}},
		FundMoves:    map[string]PriceMove{"FUND-A": {Basis: MoveBasisLastNAV, ChangePct: 50}},
		MoveIssues:   map[string]string{"ETH": "no bars"},
		Now:          time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC),
	}
}

func TestGivenHoldingsAndMovesWhenComputingExposureThenTheVolatileShareAndTodaysChangeComeFirst(t *testing.T) {
	exposure := ComputeExposure(exposureInputs())

	// volatile = funds 30M + crypto 100M of 200M
	if exposure.Total != 200_000_000 || exposure.VolatileValue != 130_000_000 || exposure.VolatileShare != 65 {
		t.Fatalf("exposure = %+v", exposure)
	}
	// BTC 70M after +40%: 20M; fund 30M after +50%: 10M
	if exposure.Change == nil || !near(*exposure.Change, 30_000_000, 0.01) || exposure.Covered != 100_000_000 {
		t.Fatalf("change = %v covered %v", exposure.Change, exposure.Covered)
	}
	if exposure.ChangePct == nil || *exposure.ChangePct != 42.86 {
		t.Fatalf("change pct = %v", exposure.ChangePct)
	}
	cash, crypto := exposure.class(AllocationCash), exposure.class(AllocationCrypto)
	if cash == nil || cash.Volatile || cash.Change != nil || cash.Share != 30 {
		t.Fatalf("cash = %+v", cash)
	}
	if crypto == nil || !crypto.Volatile || crypto.Covered != 70_000_000 || !near(*crypto.Change, 20_000_000, 0.01) || *crypto.ChangePct != 40 {
		t.Fatalf("crypto = %+v", crypto)
	}
	if lent := exposure.class(AllocationMoneyLent); lent == nil || lent.Volatile || lent.Value != 10_000_000 {
		t.Fatalf("money lent = %+v", lent)
	}
	if len(exposure.Issues) != 1 || !strings.Contains(exposure.Issues[0], "ETH") {
		t.Fatalf("issues = %q", exposure.Issues)
	}
}

func TestGivenCryptoHoldingsWhenComputingExposureThenCoinsAreSummedPerSymbolWithoutPlatforms(t *testing.T) {
	in := exposureInputs()
	extra := 5_000_000.0
	half := 0.5
	in.Valuation.Holdings = append(in.Valuation.Holdings, ValuedHolding{Section: PortfolioSectionCrypto, Holding: Holding{Platform: "Wallet C", Symbol: "BTC", Quantity: &half}, ValueVND: &extra})
	coins := ComputeExposure(in).Crypto.Coins

	if len(coins) != 3 || coins[0].Symbol != "BTC" || coins[0].Value != 75_000_000 || *coins[0].Quantity != 1.5 || coins[0].Price == nil {
		t.Fatalf("coins = %+v", coins)
	}
	if coins[0].Share != 71.43 || coins[0].Move == nil || coins[1].Symbol != "ETH" || coins[1].Issue != "no bars" || coins[1].Change != nil {
		t.Fatalf("coins = %+v", coins)
	}
	if zzz := coins[2]; zzz.Unvalued != 1 || zzz.Value != 0 || zzz.Issue == "" {
		t.Fatalf("unvalued coin = %+v", zzz)
	}
}

func TestGivenNetWorthWhenComputingExposureThenTheCryptoDrawdownScenariosUseTheFixedFalls(t *testing.T) {
	view := ComputeExposure(exposureInputs()).Crypto

	if view.Value != 100_000_000 || view.ShareOfNetWorth == nil || *view.ShareOfNetWorth != 66.67 || view.Share != 50 {
		t.Fatalf("crypto view = %+v", view)
	}
	if len(view.Scenarios) != 2 {
		t.Fatalf("scenarios = %+v", view.Scenarios)
	}
	thirty, fifty := view.Scenarios[0], view.Scenarios[1]
	if thirty.FallPct != 30 || thirty.Loss != 30_000_000 || thirty.NetWorthAfter != 120_000_000 || *thirty.NetWorthChangePct != -20 {
		t.Fatalf("30%% = %+v", thirty)
	}
	if fifty.FallPct != 50 || fifty.Loss != 50_000_000 || *fifty.NetWorthChangePct != -33.33 {
		t.Fatalf("50%% = %+v", fifty)
	}
}

func TestGivenNoPositiveNetWorthOrMovesWhenComputingExposureThenPercentagesAreLeftOut(t *testing.T) {
	in := exposureInputs()
	in.BalanceSheet.NetWorth = -1
	in.CryptoMoves, in.FundMoves, in.MoveIssues = nil, nil, nil
	exposure := ComputeExposure(in)

	if exposure.Change != nil || exposure.ChangePct != nil || exposure.Crypto.ShareOfNetWorth != nil || exposure.Crypto.Change != nil {
		t.Fatalf("exposure = %+v", exposure)
	}
	if exposure.Crypto.Scenarios[0].NetWorthChangePct != nil || exposure.Crypto.Scenarios[0].Loss != 30_000_000 {
		t.Fatalf("scenario = %+v", exposure.Crypto.Scenarios[0])
	}
	empty := ComputeExposure(ExposureInputs{})
	if empty.VolatileShare != 0 || len(empty.Classes) != 0 || empty.Crypto.Coins == nil || empty.Issues == nil {
		t.Fatalf("empty = %+v", empty)
	}
}

func TestGivenAnExposureWhenSummarisingFactsThenOnlyClassesAndScenariosAreKept(t *testing.T) {
	facts := ComputeExposure(exposureInputs()).Facts()

	if facts.VolatileSharePct != 65 || len(facts.Classes) != 4 || facts.CryptoValueVND != 100_000_000 || len(facts.CryptoDrawdowns) != 2 {
		t.Fatalf("facts = %+v", facts)
	}
	if facts.TodayChangeVND == nil || *facts.TodayChangeVND != 30_000_000 || facts.Note == "" {
		t.Fatalf("change = %v", facts.TodayChangeVND)
	}
}
