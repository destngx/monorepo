package domain

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Financial health score (WM-HEA-1..3) and freedom months (WM-RULE-12).
// Everything here is deterministic so the UI, AI and MCP agree.

const (
	healthSavingsMax         = 30
	healthEmergencyMax       = 30
	healthDebtMax            = 20
	healthDiversificationMax = 20

	TargetSavingsRate     = 30.0
	TargetEmergencyMonths = 6.0
	ReferenceDebtToIncome = 35.0

	healthLookbackMonths = 3
)

type HealthBand string

const (
	HealthBandRed    HealthBand = "red"
	HealthBandYellow HealthBand = "yellow"
	HealthBandGreen  HealthBand = "green"
)

// HealthBandFor: red < 50, yellow 50-75, green > 75.
func HealthBandFor(score float64) HealthBand {
	switch {
	case score < 50:
		return HealthBandRed
	case score <= 75:
		return HealthBandYellow
	default:
		return HealthBandGreen
	}
}

type AssetClass string

const (
	AssetClassCash       AssetClass = "cash"
	AssetClassInvestment AssetClass = "investment"
	AssetClassCrypto     AssetClass = "crypto"
)

var assetClasses = []AssetClass{AssetClassCash, AssetClassInvestment, AssetClassCrypto}

// AssetClassFor maps an account type to an asset class; liabilities and deprecated accounts have none.
func AssetClassFor(accountType AccountType) (AssetClass, bool) {
	switch accountType {
	case AccountTypeActiveUse, AccountTypeRarelyUse, AccountTypeBank, AccountTypeCash:
		return AssetClassCash, true
	case AccountTypeInvestment, AccountTypeLongHolding:
		return AssetClassInvestment, true
	case AccountTypeCrypto:
		return AssetClassCrypto, true
	default:
		return "", false
	}
}

// AssetAllocation sums positive balances per asset class.
func AssetAllocation(accounts []Account) map[AssetClass]float64 {
	allocation := map[AssetClass]float64{}
	for _, account := range accounts {
		class, ok := AssetClassFor(account.Type)
		if ok && account.Balance > 0 {
			allocation[class] += account.Balance
		}
	}
	return allocation
}

// HealthComponent is one scored pillar with its "show your math" breakdown.
type HealthComponent struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Value    float64 `json:"value"`
	Target   float64 `json:"target"`
	Unit     string  `json:"unit"`
	Score    float64 `json:"score"`
	MaxScore float64 `json:"maxScore"`
	Formula  string  `json:"formula"`
	Detail   string  `json:"detail"`
}

// HealthRecommendation is one of the "Doctor's Orders".
type HealthRecommendation struct {
	Component string `json:"component"`
	Title     string `json:"title"`
	Action    string `json:"action"`
}

type FinancialHealth struct {
	Score              float64                `json:"score"`
	Band               HealthBand             `json:"band"`
	Components         []HealthComponent      `json:"components"`
	Recommendations    []HealthRecommendation `json:"recommendations"`
	LiquidAssets       float64                `json:"liquidAssets"`
	AvgMonthlyIncome   float64                `json:"avgMonthlyIncome"`
	AvgMonthlyExpense  float64                `json:"avgMonthlyExpense"`
	NetWorth           float64                `json:"netWorth"`
	FreedomMonths      *float64               `json:"freedomMonths"`
	Allocation         map[AssetClass]float64 `json:"allocation"`
	LookbackMonths     int                    `json:"lookbackMonths"`
	InsufficientIncome bool                   `json:"insufficientIncome"`
}

// ComputeFinancialHealth scores the last full months of activity against the targets.
func ComputeFinancialHealth(accounts []Account, loans []Loan, transactions []Transaction, today time.Time) FinancialHealth {
	sheet := ComputeBalanceSheet(accounts, loans)
	allocation := AssetAllocation(accounts)
	income, expense := averageIncomeExpense(transactions, today, healthLookbackMonths)
	liquid := allocation[AssetClassCash]

	savingsRate := 0.0
	if income > 0 {
		savingsRate = round2((income - expense) / income * 100)
	}
	emergencyMonths := 0.0
	switch {
	case expense > 0:
		emergencyMonths = round2(liquid / expense)
	case liquid > 0:
		emergencyMonths = TargetEmergencyMonths
	}
	debtToIncome := 0.0
	if income > 0 {
		debtToIncome = round2(sheet.Liabilities / (income * 12) * 100)
	} else if sheet.Liabilities > 0 {
		debtToIncome = 100
	}
	hhi, diversification := diversificationScore(allocation)

	components := []HealthComponent{
		{
			Key: "savings", Label: "Savings rate", Value: savingsRate, Target: TargetSavingsRate, Unit: "%", MaxScore: healthSavingsMax,
			Score:   round2(healthSavingsMax * clampUnit(savingsRate/TargetSavingsRate)),
			Formula: "(avg income - avg expense) / avg income x 100; full score at 30%",
			Detail:  fmt.Sprintf("(%.0f - %.0f) / %.0f = %.2f%%", income, expense, income, savingsRate),
		},
		{
			Key: "emergency", Label: "Emergency fund", Value: emergencyMonths, Target: TargetEmergencyMonths, Unit: "months", MaxScore: healthEmergencyMax,
			Score:   round2(healthEmergencyMax * clampUnit(emergencyMonths/TargetEmergencyMonths)),
			Formula: "liquid assets / avg monthly expense; full score at 6 months",
			Detail:  fmt.Sprintf("%.0f / %.0f = %.2f months", liquid, expense, emergencyMonths),
		},
		{
			Key: "debt", Label: "Debt to income", Value: debtToIncome, Target: ReferenceDebtToIncome, Unit: "%", MaxScore: healthDebtMax,
			Score:   round2(healthDebtMax * clampUnit(1-debtToIncome/(2*ReferenceDebtToIncome))),
			Formula: "liabilities / annual income x 100; full score at 0%, half at 35%, zero at 70%",
			Detail:  fmt.Sprintf("%.0f / (%.0f x 12) = %.2f%%", sheet.Liabilities, income, debtToIncome),
		},
		{
			Key: "diversification", Label: "Diversification", Value: round2(hhi), Target: 1 / float64(len(assetClasses)), Unit: "HHI", MaxScore: healthDiversificationMax,
			Score:   round2(diversification),
			Formula: "Herfindahl index over cash, investment and crypto; full score when evenly split",
			Detail:  allocationDetail(allocation),
		},
	}
	total := 0.0
	for _, component := range components {
		total += component.Score
	}
	total = round2(total)

	health := FinancialHealth{
		Score: total, Band: HealthBandFor(total), Components: components, LiquidAssets: liquid,
		AvgMonthlyIncome: income, AvgMonthlyExpense: expense, NetWorth: sheet.NetWorth,
		FreedomMonths: FreedomMonths(sheet.NetWorth, expense), Allocation: allocation,
		LookbackMonths: healthLookbackMonths, InsufficientIncome: income == 0,
	}
	health.Recommendations = doctorsOrders(components, health)
	return health
}

func averageIncomeExpense(transactions []Transaction, today time.Time, months int) (float64, float64) {
	current := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	flows := MonthlyCashFlows(transactions, current.AddDate(0, -months, 0), current.AddDate(0, -1, 0))
	var income, expense float64
	for _, flow := range flows {
		income += flow.Income
		expense += flow.Expense
	}
	return math.Round(income / float64(months)), math.Round(expense / float64(months))
}

// diversificationScore returns the HHI and its score: 1/N (even) scores full, 1 (single class) scores zero.
func diversificationScore(allocation map[AssetClass]float64) (float64, float64) {
	total := 0.0
	for _, value := range allocation {
		total += value
	}
	if total <= 0 {
		return 1, 0
	}
	hhi := 0.0
	for _, value := range allocation {
		share := value / total
		hhi += share * share
	}
	minimum := 1 / float64(len(assetClasses))
	return hhi, healthDiversificationMax * clampUnit((1-hhi)/(1-minimum))
}

func allocationDetail(allocation map[AssetClass]float64) string {
	total := 0.0
	for _, value := range allocation {
		total += value
	}
	detail := ""
	for _, class := range assetClasses {
		share := 0.0
		if total > 0 {
			share = allocation[class] / total * 100
		}
		if detail != "" {
			detail += ", "
		}
		detail += fmt.Sprintf("%s %.1f%%", class, share)
	}
	return detail
}

// doctorsOrders returns three actions for the weakest pillars (lowest score share first).
func doctorsOrders(components []HealthComponent, health FinancialHealth) []HealthRecommendation {
	ranked := append([]HealthComponent(nil), components...)
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].Score/ranked[i].MaxScore < ranked[j].Score/ranked[j].MaxScore
	})
	orders := make([]HealthRecommendation, 0, 3)
	for _, component := range ranked[:3] {
		orders = append(orders, recommendationFor(component, health))
	}
	return orders
}

func recommendationFor(component HealthComponent, health FinancialHealth) HealthRecommendation {
	full := component.Score >= component.MaxScore
	switch component.Key {
	case "savings":
		if full {
			return HealthRecommendation{component.Key, "Keep saving", fmt.Sprintf("Savings rate %.1f%% meets the 30%% target. Route the surplus to goals or investments.", component.Value)}
		}
		gap := math.Max(0, TargetSavingsRate/100*health.AvgMonthlyIncome-(health.AvgMonthlyIncome-health.AvgMonthlyExpense))
		return HealthRecommendation{component.Key, "Raise your savings rate", fmt.Sprintf("Savings rate is %.1f%%. Cut about %.0f per month from expenses to reach 30%%.", component.Value, gap)}
	case "emergency":
		if full {
			return HealthRecommendation{component.Key, "Emergency fund is ready", fmt.Sprintf("%.1f months of expenses are liquid. Extra cash can be invested.", component.Value)}
		}
		gap := math.Max(0, TargetEmergencyMonths*health.AvgMonthlyExpense-health.LiquidAssets)
		return HealthRecommendation{component.Key, "Build the emergency fund", fmt.Sprintf("Liquid cash covers %.1f months. Add %.0f to reach 6 months.", component.Value, gap)}
	case "debt":
		if full {
			return HealthRecommendation{component.Key, "Stay debt free", "No outstanding liabilities. Keep paying cards in full each cycle."}
		}
		return HealthRecommendation{component.Key, "Pay down debt", fmt.Sprintf("Debt is %.1f%% of annual income (reference 35%%). Clear card balances first, then loans.", component.Value)}
	default:
		if full {
			return HealthRecommendation{component.Key, "Well diversified", "Assets are spread evenly across cash, investments and crypto."}
		}
		return HealthRecommendation{component.Key, "Diversify assets", fmt.Sprintf("Allocation is %s. Move surplus cash beyond the emergency fund into under-weighted classes.", component.Detail)}
	}
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

func clampUnit(value float64) float64 {
	return math.Max(0, math.Min(1, value))
}
