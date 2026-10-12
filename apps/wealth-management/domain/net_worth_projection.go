package domain

import (
	"math"
)

// ProjectionYears are the net worth projection horizons (WM-RULE-12).
var ProjectionYears = []int{5, 10, 20}

// NetWorthProjection is net worth after Years at the fallback return,
// before (Nominal) and after (Real) VND inflation (WM-RULE-12).
type NetWorthProjection struct {
	Years   int     `json:"years"`
	Nominal float64 `json:"nominal"`
	Real    float64 `json:"real"`
}

// ProjectNetWorth projects net worth over ProjectionYears at the 8% fallback
// return; Real deducts VND inflation via ProjectRealValue (WM-RULE-12).
func ProjectNetWorth(netWorth float64) []NetWorthProjection {
	projection := make([]NetWorthProjection, 0, len(ProjectionYears))
	for _, years := range ProjectionYears {
		projection = append(projection, NetWorthProjection{
			Years:   years,
			Nominal: math.Round(netWorth * math.Pow(1+DefaultAnnualReturn/100, float64(years))),
			Real:    ProjectRealValue(netWorth, 0, CurrencyVND, float64(years)),
		})
	}
	return projection
}

// FreedomMonths = net worth / monthly expenses; nil when expenses are unknown.
func FreedomMonths(netWorth float64, monthlyExpense float64) *float64 {
	if monthlyExpense <= 0 {
		return nil
	}
	months := round2(netWorth / monthlyExpense)
	return &months
}

const DefaultAnnualReturn = 8.0

// InflationFor returns the assumed annual inflation per currency (VND 6%, USD 2%).
func InflationFor(currency Currency) float64 {
	if currency == CurrencyUSD || currency == CurrencyUSDT {
		return 2
	}
	return 6
}

// ProjectRealValue = PV x (1 + r)^years with r = return - inflation (percent);
// a zero return uses the 8% fallback.
func ProjectRealValue(presentValue float64, annualReturn float64, currency Currency, years float64) float64 {
	if annualReturn == 0 {
		annualReturn = DefaultAnnualReturn
	}
	rate := (annualReturn - InflationFor(currency)) / 100
	return math.Round(presentValue * math.Pow(1+rate, years))
}
