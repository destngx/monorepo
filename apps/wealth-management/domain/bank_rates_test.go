package domain

import (
	"fmt"
	"strings"
	"testing"
)

func months(value int) *int { return &value }

// tenoredRates is the market server's unfiltered shape: several rows per
// bank (no-term, 1, 3, 6, 12, 24 and 36 months, counter and online), about
// 200 rows, with invented banks and rates.
func tenoredRates() []BankRate {
	rates := []BankRate{}
	tenors := []int{0, 1, 3, 6, 12, 24, 36}
	for bank := 1; bank <= 15; bank++ {
		for _, channel := range []string{"counter", "online"} {
			for _, tenor := range tenors {
				rate := 0.1 + float64(tenor)*0.4 + float64(bank)*0.01
				if channel == "online" {
					rate += 0.05
				}
				text := fmt.Sprintf("%dm", tenor)
				if tenor == 0 {
					text, rate = "no-term", 0.1+float64(bank)*0.01
				}
				rates = append(rates, BankRate{Bank: fmt.Sprintf("Bank %02d", bank), Tenor: text, TenorMonths: months(tenor),
					InstantAccess: tenor == 0, Channel: channel, Rate: rate, Currency: "VND"})
			}
		}
	}
	return rates
}

func TestGivenTheTenoredShapeWhenChoosingTheDepositRateThenATrue12MonthRowWins(t *testing.T) {
	rates := tenoredRates()
	if len(rates) < 200 {
		t.Fatalf("fixture has %d rows", len(rates))
	}
	// A high no-term row (digits nowhere in its tenor text) must not win the 12-month comparison.
	rates = append(rates, BankRate{Bank: "Bank 99", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 9})
	rate, ok := DepositRateFromBankRates(rates)
	if !ok || rate.Bank != "Bank 15" || rate.Tenor != "12m" || rate.TenorMonths == nil || *rate.TenorMonths != 12 || rate.Channel != "online" {
		t.Fatalf("rate = %+v", rate)
	}
	if !strings.Contains(rate.Basis, "12-month") {
		t.Fatalf("basis = %q", rate.Basis)
	}
}

func TestGivenOnlyNoTermAndUntenoredRowsWhenChoosingTheDepositRateThenTheUntenoredRateIsTheFallback(t *testing.T) {
	rate, ok := DepositRateFromBankRates([]BankRate{
		{Bank: "Bank A", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 0.5},
		{Bank: "Bank B", Tenor: "", Rate: 4.9},
	})
	if !ok || rate.Bank != "Bank B" || !strings.Contains(rate.Basis, "no tenor") {
		t.Fatalf("rate = %+v", rate)
	}
	if _, ok := DepositRateFromBankRates([]BankRate{{Bank: "Bank A", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 0.5}}); ok {
		t.Fatal("a no-term rate is not a 12-month comparison")
	}
}

func TestGivenTenoredRowsWithoutTenorMonthsWhenChoosingTheDepositRateThenTheTenorTextIsRead(t *testing.T) {
	rate, ok := DepositRateFromBankRates([]BankRate{
		{Bank: "Bank A", Tenor: "không kỳ hạn", Rate: 8},
		{Bank: "Bank B", Tenor: "12M", Rate: 4.8},
	})
	if !ok || rate.Bank != "Bank B" {
		t.Fatalf("rate = %+v", rate)
	}
}

func TestGivenTheTenoredShapeWhenChoosingTheInstantAccessRateThenTheBestOfNoTermAnd1MonthWins(t *testing.T) {
	rate, ok := InstantAccessRateFromBankRates(tenoredRates())
	if !ok || rate.Bank != "Bank 15" || rate.Access != DepositOneMonth || rate.TenorMonths == nil || *rate.TenorMonths != 1 {
		t.Fatalf("rate = %+v", rate)
	}
	if !strings.Contains(rate.AccessLabel, "rolls monthly") || !strings.Contains(rate.AccessLabel, "no-term rate") {
		t.Fatalf("label = %q", rate.AccessLabel)
	}
}

func TestGivenANoTermRateAboveEvery1MonthRateWhenChoosingTheInstantAccessRateThenNoTermWins(t *testing.T) {
	rate, ok := InstantAccessRateFromBankRates([]BankRate{
		{Bank: "Bank A", Tenor: "1m", TenorMonths: months(1), Rate: 3.9},
		{Bank: "Bank B", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 4.1},
		{Bank: "Bank C", Tenor: "12m", TenorMonths: months(12), Rate: 7.2},
		{Bank: "Bank D", Tenor: "", Rate: 7.5},
	})
	if !ok || rate.Bank != "Bank B" || rate.Access != DepositNoTerm || rate.AccessLabel != "no-term, withdraw any time" {
		t.Fatalf("rate = %+v", rate)
	}
}

func TestGivenEqualNoTermAnd1MonthRatesWhenChoosingTheInstantAccessRateThenNoTermWins(t *testing.T) {
	rate, ok := InstantAccessRateFromBankRates([]BankRate{
		{Bank: "Bank A", Tenor: "1m", TenorMonths: months(1), Rate: 4},
		{Bank: "Bank B", Tenor: "no-term", TenorMonths: months(0), InstantAccess: true, Rate: 4},
	})
	if !ok || rate.Access != DepositNoTerm {
		t.Fatalf("rate = %+v", rate)
	}
}

func TestGivenOnlyUntenoredOrLongRatesWhenChoosingTheInstantAccessRateThenThereIsNone(t *testing.T) {
	if rate, ok := InstantAccessRateFromBankRates([]BankRate{{Bank: "Bank A", Rate: 7.2}, {Bank: "Bank B", Tenor: "6m", TenorMonths: months(6), Rate: 6}}); ok {
		t.Fatalf("an untenored or long-term rate must never stand in, got %+v", rate)
	}
}

func TestGivenRatesWhenLabellingThemThenNoEmptyParenthesesAppear(t *testing.T) {
	for _, tc := range []struct {
		rate DepositRate
		want string
	}{
		{DepositRate{Bank: "Bank A", Rate: 7.2}, "7.20%"},
		{DepositRate{Bank: "Bank A", Rate: 7.2, Tenor: "12M"}, "7.20%, 12M"},
		{DepositRate{Bank: "Bank A", Rate: 7.2, Tenor: "12m", TenorMonths: months(12)}, "7.20%, 12-month"},
		{DepositRate{Bank: "Bank A", Rate: 0.5, Tenor: "no-term", TenorMonths: months(0)}, "0.50%, no-term"},
	} {
		if got := tc.rate.Label(); got != tc.want {
			t.Fatalf("Label(%+v) = %q, want %q", tc.rate, got, tc.want)
		}
	}
}
